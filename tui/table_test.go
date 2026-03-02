//go:build linux

package tui

import (
	"testing"
	"time"

	"github.com/fabioconcina/arpdvark/state"
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
	devices := []state.Device{
		{
			IP:       "192.168.1.1",
			MAC:      "aa:bb:cc:dd:ee:ff",
			Hostname: "router.home",
			Vendor:   "Acme Corp",
			Online:   true,
		},
		{
			IP:       "192.168.1.2",
			MAC:      "11:22:33:44:55:66",
			Hostname: "",
			Vendor:   "Unknown",
			Online:   true,
		},
	}
	rows := devicesToRows(devices, nil)
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
	// index 3 = Label (empty, no tags passed)
	if rows[0][3] != "" {
		t.Errorf("rows[0][3] (label) = %q, want \"\"", rows[0][3])
	}
	// index 4 = Vendor
	if rows[0][4] != "Acme Corp" {
		t.Errorf("rows[0][4] = %q, want %q", rows[0][4], "Acme Corp")
	}

	if rows[1][2] != "" {
		t.Errorf("rows[1][2] (empty hostname) = %q, want %q", rows[1][2], "")
	}
}

func TestDevicesToRows_Empty(t *testing.T) {
	rows := devicesToRows(nil, nil)
	if len(rows) != 0 {
		t.Errorf("devicesToRows(nil) = %d rows, want 0", len(rows))
	}
}

func TestDevicesToRows_WithLabels(t *testing.T) {
	devices := []state.Device{
		{IP: "192.168.1.1", MAC: "aa:bb:cc:dd:ee:ff", Hostname: "router", Vendor: "Acme", Online: true},
	}
	labels := map[string]string{"aa:bb:cc:dd:ee:ff": "my-router"}
	rows := devicesToRows(devices, labels)
	if rows[0][3] != "my-router" {
		t.Errorf("label col = %q, want %q", rows[0][3], "my-router")
	}
	if rows[0][4] != "Acme" {
		t.Errorf("vendor col = %q, want %q", rows[0][4], "Acme")
	}
}

func TestDevicesToRows_NoLabel(t *testing.T) {
	devices := []state.Device{
		{IP: "192.168.1.2", MAC: "11:22:33:44:55:66", Online: true},
	}
	rows := devicesToRows(devices, nil)
	if rows[0][3] != "" {
		t.Errorf("label col = %q, want empty", rows[0][3])
	}
}

func TestDevicesToRows_OfflineDimmed(t *testing.T) {
	devices := []state.Device{
		{IP: "192.168.1.1", MAC: "aa:bb:cc:dd:ee:ff", Vendor: "Acme", Online: false},
	}
	rows := devicesToRows(devices, nil)
	// Offline rows should contain ANSI escape sequences from the dim style.
	if rows[0][0] == "192.168.1.1" {
		t.Error("offline device IP should be styled (contain ANSI escapes)")
	}
}

func TestCountOnline(t *testing.T) {
	devices := []state.Device{
		{Online: true},
		{Online: false},
		{Online: true},
	}
	if got := countOnline(devices); got != 2 {
		t.Errorf("countOnline = %d, want 2", got)
	}
}
