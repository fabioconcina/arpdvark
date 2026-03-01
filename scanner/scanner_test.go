//go:build linux

package scanner

import (
	"net"
	"testing"
)

func TestHostsInSubnet_Count(t *testing.T) {
	tests := []struct {
		cidr  string
		count int
	}{
		{"192.168.1.0/30", 2},
		{"192.168.1.0/29", 6},
		{"192.168.1.0/28", 14},
		{"10.0.0.0/24", 254},
	}
	for _, tt := range tests {
		t.Run(tt.cidr, func(t *testing.T) {
			_, subnet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("invalid CIDR %s: %v", tt.cidr, err)
			}
			hosts := hostsInSubnet(subnet)
			if len(hosts) != tt.count {
				t.Errorf("hostsInSubnet(%s) = %d hosts, want %d", tt.cidr, len(hosts), tt.count)
			}
		})
	}
}

func TestHostsInSubnet_Boundaries(t *testing.T) {
	_, subnet, _ := net.ParseCIDR("192.168.1.0/30")
	hosts := hostsInSubnet(subnet)
	// /30 has exactly 2 usable hosts: .1 and .2 (not .0 network or .3 broadcast)
	if hosts[0].String() != "192.168.1.1" {
		t.Errorf("first host = %s, want 192.168.1.1", hosts[0])
	}
	if hosts[1].String() != "192.168.1.2" {
		t.Errorf("last host = %s, want 192.168.1.2", hosts[1])
	}
}

func TestSortedDevices(t *testing.T) {
	seen := map[string]Device{
		"10.0.0.3": {IP: net.ParseIP("10.0.0.3").To4()},
		"10.0.0.1": {IP: net.ParseIP("10.0.0.1").To4()},
		"10.0.0.2": {IP: net.ParseIP("10.0.0.2").To4()},
	}
	sorted := sortedDevices(seen)
	expected := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
	if len(sorted) != len(expected) {
		t.Fatalf("sortedDevices() returned %d devices, want %d", len(sorted), len(expected))
	}
	for i, d := range sorted {
		if d.IP.String() != expected[i] {
			t.Errorf("sorted[%d] = %s, want %s", i, d.IP, expected[i])
		}
	}
}

func TestSortedDevices_Empty(t *testing.T) {
	result := sortedDevices(map[string]Device{})
	if len(result) != 0 {
		t.Errorf("sortedDevices(empty) returned %d devices, want 0", len(result))
	}
}
