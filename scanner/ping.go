package scanner

import (
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	pingTimeout  = 1 * time.Second
	pingProtocol = 1 // ICMP for IPv4
)

// pingDevices sends ICMP echo requests to all IPs and returns a map of IP → RTT.
// Devices that don't respond within the timeout are omitted from the result.
func pingDevices(devices []Device) map[string]time.Duration {
	results := make(map[string]time.Duration)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, d := range devices {
		wg.Add(1)
		go func(ip net.IP, seq int) {
			defer wg.Done()
			rtt, ok := pingOne(ip, seq)
			if ok {
				mu.Lock()
				results[ip.String()] = rtt
				mu.Unlock()
			}
		}(d.IP, i)
	}
	wg.Wait()
	return results
}

// pingOne sends a single ICMP echo request and waits for a reply.
func pingOne(ip net.IP, seq int) (time.Duration, bool) {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return 0, false
	}
	defer conn.Close()

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   seq & 0xffff,
			Seq:  seq & 0xffff,
			Data: []byte("arpdvark"),
		},
	}
	wb, err := msg.Marshal(nil)
	if err != nil {
		return 0, false
	}

	dst := &net.IPAddr{IP: ip}
	conn.SetDeadline(time.Now().Add(pingTimeout))

	start := time.Now()
	if _, err := conn.WriteTo(wb, dst); err != nil {
		return 0, false
	}

	buf := make([]byte, 512)
	for {
		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			return 0, false
		}
		rm, err := icmp.ParseMessage(pingProtocol, buf[:n])
		if err != nil {
			continue
		}
		if rm.Type != ipv4.ICMPTypeEchoReply {
			continue
		}
		// Verify this reply is from the target IP.
		if peerIP, ok := peer.(*net.IPAddr); ok && peerIP.IP.Equal(ip) {
			return time.Since(start), true
		}
	}
}
