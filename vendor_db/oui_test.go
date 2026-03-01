package vendor_db

import (
	"net"
	"testing"
)

func TestLookup_KnownVendor(t *testing.T) {
	// 28:6F:B9 is Nokia Shanghai Bell Co., Ltd. in the embedded oui.txt
	mac := net.HardwareAddr{0x28, 0x6F, 0xB9, 0x00, 0x00, 0x01}
	got := Lookup(mac)
	want := "Nokia Shanghai Bell Co., Ltd."
	if got != want {
		t.Errorf("Lookup(%v) = %q, want %q", mac, got, want)
	}
}

func TestLookup_Unknown(t *testing.T) {
	// FC:FF:FF is not a real OUI assignment and won't be in any vendor DB
	mac := net.HardwareAddr{0xFC, 0xFF, 0xFF, 0x00, 0x00, 0x00}
	got := Lookup(mac)
	if got != "Unknown" {
		t.Errorf("Lookup(%v) = %q, want %q", mac, got, "Unknown")
	}
}

func TestLookup_LocallyAdministered(t *testing.T) {
	tests := []struct {
		name string
		mac  net.HardwareAddr
	}{
		{"bit set", net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{"random mac", net.HardwareAddr{0xEA, 0x03, 0x65, 0x53, 0xC9, 0x62}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Lookup(tt.mac)
			if got != "Local/Randomized" {
				t.Errorf("Lookup(%v) = %q, want %q", tt.mac, got, "Local/Randomized")
			}
		})
	}
}

func TestLookup_TooShort(t *testing.T) {
	tests := []struct {
		name string
		mac  net.HardwareAddr
	}{
		{"nil", nil},
		{"one byte", net.HardwareAddr{0x28}},
		{"two bytes", net.HardwareAddr{0x28, 0x6F}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Lookup(tt.mac)
			if got != "Unknown" {
				t.Errorf("Lookup(%v) = %q, want %q", tt.mac, got, "Unknown")
			}
		})
	}
}

func TestDBPopulated(t *testing.T) {
	if len(db) == 0 {
		t.Error("OUI database is empty; oui.txt may be missing or malformed")
	}
}
