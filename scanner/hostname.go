package scanner

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/hex"
	"net"
	"os"
	"strings"
	"time"
)

// gatewayIP returns the default gateway IP by reading the Linux routing table.
// Falls back to the first host in the subnet if the routing table is unavailable.
func gatewayIP(subnet *net.IPNet) string {
	if gw := gatewayFromRoute(); gw != "" {
		return gw
	}
	base := binary.BigEndian.Uint32(subnet.IP.To4())
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, base+1)
	return ip.String()
}

// gatewayFromRoute reads /proc/net/route and returns the default gateway IP.
// Returns "" if the file cannot be read or no default route is found.
func gatewayFromRoute() string {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip header line
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		// Default route: Destination == "00000000"
		if fields[1] != "00000000" {
			continue
		}
		// Gateway is a little-endian hex-encoded 32-bit IP.
		gwHex := fields[2]
		b, err := hex.DecodeString(gwHex)
		if err != nil || len(b) != 4 {
			continue
		}
		// /proc/net/route stores IPs in native (little-endian on x86/ARM) byte order.
		return net.IPv4(b[3], b[2], b[1], b[0]).String()
	}
	return ""
}

// lookupHostname resolves an IP to a hostname by trying three methods in order:
//  1. System resolver (uses /etc/resolv.conf)
//  2. Gateway as DNS server (dnsmasq on home routers has PTR records for DHCP leases)
//  3. mDNS unicast to the device's port 5353 (Bonjour/Avahi)
func lookupHostname(ctx context.Context, ip, gateway string) string {
	ctx1, cancel1 := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel1()
	if h := resolveWith(ctx1, ip, ""); h != "" {
		return h
	}

	if gateway != "" {
		ctx2, cancel2 := context.WithTimeout(ctx, 800*time.Millisecond)
		defer cancel2()
		if h := resolveWith(ctx2, ip, gateway+":53"); h != "" {
			return h
		}
	}

	ctx3, cancel3 := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel3()
	if h := resolveWith(ctx3, ip, ip+":5353"); h != "" {
		return h
	}

	return ""
}

// resolveWith does a reverse DNS lookup using the system resolver (server=="")
// or a custom resolver pointing at the given host:port.
func resolveWith(ctx context.Context, ip, server string) string {
	var r *net.Resolver
	if server == "" {
		r = net.DefaultResolver
	} else {
		r = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 500 * time.Millisecond}).DialContext(ctx, "udp", server)
			},
		}
	}
	names, err := r.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}
