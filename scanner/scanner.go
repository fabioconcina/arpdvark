package scanner

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/fabioconcina/arpdvark/vendor_db"

	"golang.org/x/time/rate"
)

// Device represents a discovered network host.
type Device struct {
	IP        net.IP
	MAC       net.HardwareAddr
	Vendor    string
	Hostname  string
	FirstSeen time.Time
	LastSeen  time.Time

}

// Scanner performs ARP scans on a network interface.
type Scanner struct {
	iface      *net.Interface
	srcIP      net.IP
	subnet     *net.IPNet
	allowLarge bool

	mu       sync.Mutex
	seen     map[string]Device
	scanNum  int
}

// New creates a Scanner for the given interface name.
// If ifaceName is empty, the first suitable interface is auto-detected.
func New(ifaceName string, allowLarge bool) (*Scanner, error) {
	var iface *net.Interface
	var err error

	if ifaceName != "" {
		iface, err = net.InterfaceByName(ifaceName)
		if err != nil {
			return nil, fmt.Errorf("interface %q not found: %w", ifaceName, err)
		}
	} else {
		iface, err = autoDetectInterface()
		if err != nil {
			return nil, err
		}
	}

	srcIP, subnet, err := ifaceIPNet(iface)
	if err != nil {
		return nil, fmt.Errorf("interface %s has no usable IPv4 address: %w", iface.Name, err)
	}

	ones, _ := subnet.Mask.Size()
	if ones < 8 {
		return nil, fmt.Errorf("subnet %s is too large (smaller than /8); arpdvark only supports /8–/28", subnet)
	}
	if ones > 28 {
		return nil, fmt.Errorf("subnet %s is too small (larger than /28)", subnet)
	}
	if ones < 16 && !allowLarge {
		return nil, fmt.Errorf(
			"subnet %s has more than 65534 hosts; pass --large to scan anyway", subnet,
		)
	}

	// Preflight: verify we can open a raw ARP socket.
	if c, err := dialARP(iface); err != nil {
		return nil, fmt.Errorf("cannot open ARP socket (try running with sudo): %w", err)
	} else {
		c.Close()
	}

	return &Scanner{
		iface:      iface,
		srcIP:      srcIP,
		subnet:     subnet,
		allowLarge: allowLarge,
		seen:       make(map[string]Device),
	}, nil
}

// Interface returns the name of the network interface being scanned.
func (s *Scanner) Interface() string { return s.iface.Name }

// Subnet returns the CIDR notation of the subnet being scanned.
func (s *Scanner) Subnet() string { return s.subnet.String() }

// Scan performs one ARP sweep and returns the merged device list.
func (s *Scanner) Scan(ctx context.Context) ([]Device, error) {
	// Use separate clients for sending and receiving to avoid
	// concurrent read/write on a single arp.Client.
	recvClient, err := dialARP(s.iface)
	if err != nil {
		return nil, fmt.Errorf("cannot open ARP socket (try running with sudo): %w", err)
	}
	defer recvClient.Close()

	sendClient, err := dialARP(s.iface)
	if err != nil {
		return nil, fmt.Errorf("cannot open ARP socket (try running with sudo): %w", err)
	}
	defer sendClient.Close()

	hosts := hostsInSubnet(s.subnet)

	// Start collecting replies; it runs until we close doneCh.
	doneCh := make(chan struct{})
	replyCh := make(chan []arpReply, 1)
	go func() {
		replyCh <- collectReplies(recvClient, s.subnet, doneCh)
	}()

	// Rate-limit sends: 1000 pkt/s for small subnets, 5000 for larger ones.
	ones, _ := s.subnet.Mask.Size()
	pps := rate.Limit(1000)
	if ones < 24 {
		pps = 5000
	}
	limiter := rate.NewLimiter(pps, int(pps))

	workers := len(hosts)
	if workers > 256 {
		workers = 256
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	// Send multiple rounds on the first scan to catch slow responders;
	// subsequent scans only need one round since seen accumulates.
	s.mu.Lock()
	s.scanNum++
	rounds := 1
	if s.scanNum == 1 {
		rounds = 3
	}
	s.mu.Unlock()
outer:
	for round := 0; round < rounds; round++ {
		if round > 0 {
			select {
			case <-ctx.Done():
				break outer
			case <-time.After(100 * time.Millisecond):
			}
		}
		for _, ip := range hosts {
			if err := limiter.Wait(ctx); err != nil {
				break outer
			}
			select {
			case <-ctx.Done():
				break outer
			case sem <- struct{}{}:
			}
			wg.Add(1)
			go func(target net.IP) {
				defer wg.Done()
				defer func() { <-sem }()
				_ = sendARP(sendClient, target, s.srcIP, s.iface.HardwareAddr)
			}(ip)
		}
		wg.Wait()
	}
	wg.Wait()

	// Give stragglers 2 seconds to reply, then stop the collector.
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
	}
	close(doneCh)
	replies := <-replyCh

	// Resolve hostnames and ping devices concurrently.
	type dnsResult struct {
		ip       string
		hostname string
	}
	dnsCtx, dnsCancel := context.WithTimeout(ctx, 2*time.Second)
	defer dnsCancel()
	gateway := gatewayIP(s.subnet)
	dnsCh := make(chan dnsResult, len(replies))
	for _, r := range replies {
		ipStr := r.IP.String()
		go func(ipStr string) {
			dnsCh <- dnsResult{ip: ipStr, hostname: lookupHostname(dnsCtx, ipStr, gateway)}
		}(ipStr)
	}

	hostnames := make(map[string]string, len(replies))
	for range replies {
		res := <-dnsCh
		hostnames[res.ip] = res.hostname
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range replies {
		key := r.IP.String()
		existing, ok := s.seen[key]
		if !ok {
			existing = Device{
				IP:        r.IP,
				MAC:       r.MAC,
				Vendor:    vendor_db.Lookup(r.MAC),
				FirstSeen: now,
			}
		} else if !bytes.Equal(existing.MAC, r.MAC) {
			existing.MAC = r.MAC
			existing.Vendor = vendor_db.Lookup(r.MAC)
		}
		existing.Hostname = hostnames[key]
		existing.LastSeen = now
		s.seen[key] = existing
	}

	return sortedDevices(s.seen), nil
}

func autoDetectInterface() (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for i := range ifaces {
		iface := &ifaces[i]
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil {
				return iface, nil
			}
		}
	}
	return nil, fmt.Errorf("no suitable network interface found; use -i to specify one")
}

func ifaceIPNet(iface *net.Interface) (net.IP, *net.IPNet, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, nil, err
	}
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip4 := ipnet.IP.To4()
		if ip4 == nil {
			continue
		}
		subnet := &net.IPNet{
			IP:   ip4.Mask(ipnet.Mask),
			Mask: ipnet.Mask,
		}
		return ip4, subnet, nil
	}
	return nil, nil, fmt.Errorf("no IPv4 address")
}

func hostsInSubnet(subnet *net.IPNet) []net.IP {
	base := binary.BigEndian.Uint32(subnet.IP.To4())
	mask := binary.BigEndian.Uint32(subnet.Mask)
	size := ^mask

	var hosts []net.IP
	for i := uint32(1); i < size; i++ {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, base+i)
		hosts = append(hosts, ip)
	}
	return hosts
}

func sortedDevices(seen map[string]Device) []Device {
	devices := make([]Device, 0, len(seen))
	for _, d := range seen {
		devices = append(devices, d)
	}
	sort.Slice(devices, func(i, j int) bool {
		a := binary.BigEndian.Uint32(devices[i].IP.To4())
		b := binary.BigEndian.Uint32(devices[j].IP.To4())
		return a < b
	})
	return devices
}
