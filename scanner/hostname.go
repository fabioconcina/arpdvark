package scanner

import (
	"context"
	"encoding/binary"
	"net"
	"strings"
	"time"
)

// gatewayIP returns the first host in the subnet (conventionally the router).
func gatewayIP(subnet *net.IPNet) string {
	base := binary.BigEndian.Uint32(subnet.IP.To4())
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, base+1)
	return ip.String()
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
