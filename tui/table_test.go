//go:build linux

package tui

import (
	"net"
	"testing"
	"time"

	"github.com/fabioconcina/arpdvark/scanner"
)

func TestHumanDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Second, "5s"},
		{59 * time.Second, "59s"},
		{90 * time.Second, "1m30s"},
		{time.Hour + 30*time.Minute, "1h30m"},
		{2*time.Hour + 5*time.Minute, "2h5m"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := humanDuration(tt.d)
			if got != tt.want {
				t.Errorf("humanDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestDevicesToRows(t *testing.T) {
	devices := []scanner.Device{
		{
			IP:       net.ParseIP("192.168.1.1").To4(),
			MAC:      net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
			Hostname: "router.home",
			Vendor:   "Acme Corp",
		},
		{
			IP:       net.ParseIP("192.168.1.2").To4(),
			MAC:      net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
			Hostname: "",
			Vendor:   "Unknown",
		},
	}
	rows := devicesToRows(devices)
	if len(rows) != 2 {
		t.Fatalf("devicesToRows() = %d rows, want 2", len(rows))
	}

	if rows[0][0] != "192.168.1.1" {
		t.Errorf("rows[0][0] = %q, want %q", rows[0][0], "192.168.1.1")
	}
	if rows[0][1] != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("rows[0][1] = %q, want %q", rows[0][1], "aa:bb:cc:dd:ee:ff")
	}
	if rows[0][2] != "router.home" {
		t.Errorf("rows[0][2] = %q, want %q", rows[0][2], "router.home")
	}
	if rows[0][3] != "Acme Corp" {
		t.Errorf("rows[0][3] = %q, want %q", rows[0][3], "Acme Corp")
	}

	if rows[1][2] != "" {
		t.Errorf("rows[1][2] (empty hostname) = %q, want %q", rows[1][2], "")
	}
}

func TestDevicesToRows_Empty(t *testing.T) {
	rows := devicesToRows(nil)
	if len(rows) != 0 {
		t.Errorf("devicesToRows(nil) = %d rows, want 0", len(rows))
	}
}
