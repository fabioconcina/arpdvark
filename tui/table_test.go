//go:build linux

package tui

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fabioconcina/arpdvark/state"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}

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
	rows := devicesToRows(devices, nil, nil)
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
	rows := devicesToRows(nil, nil, nil)
	if len(rows) != 0 {
		t.Errorf("devicesToRows(nil) = %d rows, want 0", len(rows))
	}
}

func TestDevicesToRows_WithLabels(t *testing.T) {
	devices := []state.Device{
		{IP: "192.168.1.1", MAC: "aa:bb:cc:dd:ee:ff", Hostname: "router", Vendor: "Acme", Online: true},
	}
	labels := map[string]string{"aa:bb:cc:dd:ee:ff": "my-router"}
	rows := devicesToRows(devices, labels, nil)
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
	rows := devicesToRows(devices, nil, nil)
	if rows[0][3] != "" {
		t.Errorf("label col = %q, want empty", rows[0][3])
	}
}

func TestDevicesToRows_OfflineDimmed(t *testing.T) {
	devices := []state.Device{
		{IP: "192.168.1.1", MAC: "aa:bb:cc:dd:ee:ff", Vendor: "Acme", Online: false},
	}
	rows := devicesToRows(devices, nil, nil)
	// Offline rows should contain ANSI escape sequences from the dim style.
	if rows[0][0] == "192.168.1.1" {
		t.Error("offline device IP should be styled (contain ANSI escapes)")
	}
}

func TestDevicesToRows_NewDeviceStyled(t *testing.T) {
	devices := []state.Device{
		{IP: "192.168.1.1", MAC: "aa:bb:cc:dd:ee:ff", Vendor: "Acme", Online: true},
		{IP: "192.168.1.2", MAC: "11:22:33:44:55:66", Vendor: "Other", Online: true},
	}
	newMACs := map[string]bool{"aa:bb:cc:dd:ee:ff": true}
	rows := devicesToRows(devices, nil, newMACs)
	// New device rows should contain ANSI escape sequences from the new style.
	if rows[0][0] == "192.168.1.1" {
		t.Error("new device IP should be styled (contain ANSI escapes)")
	}
	// Non-new device should be plain.
	if rows[1][0] != "192.168.1.2" {
		t.Errorf("non-new device IP = %q, want plain %q", rows[1][0], "192.168.1.2")
	}
}

func TestSortDevices_IP(t *testing.T) {
	devices := []state.Device{
		{IP: "192.168.1.10", MAC: "aa:bb:cc:dd:ee:ff"},
		{IP: "192.168.1.2", MAC: "11:22:33:44:55:66"},
		{IP: "192.168.1.1", MAC: "00:11:22:33:44:55"},
	}
	sortDevices(devices, SortIP, true, nil)
	if devices[0].IP != "192.168.1.1" || devices[1].IP != "192.168.1.2" || devices[2].IP != "192.168.1.10" {
		t.Errorf("sort by IP asc: got %s, %s, %s", devices[0].IP, devices[1].IP, devices[2].IP)
	}
	sortDevices(devices, SortIP, false, nil)
	if devices[0].IP != "192.168.1.10" {
		t.Errorf("sort by IP desc: first = %s, want 192.168.1.10", devices[0].IP)
	}
}

func TestSortDevices_Hostname(t *testing.T) {
	devices := []state.Device{
		{IP: "192.168.1.1", Hostname: "charlie"},
		{IP: "192.168.1.2", Hostname: "alpha"},
		{IP: "192.168.1.3", Hostname: "bravo"},
	}
	sortDevices(devices, SortHostname, true, nil)
	if devices[0].Hostname != "alpha" || devices[1].Hostname != "bravo" || devices[2].Hostname != "charlie" {
		t.Errorf("sort by hostname asc: got %s, %s, %s", devices[0].Hostname, devices[1].Hostname, devices[2].Hostname)
	}
}

func TestSortDevices_Label(t *testing.T) {
	devices := []state.Device{
		{IP: "192.168.1.1", MAC: "aa:bb:cc:dd:ee:ff"},
		{IP: "192.168.1.2", MAC: "11:22:33:44:55:66"},
	}
	tags := map[string]string{
		"aa:bb:cc:dd:ee:ff": "zebra",
		"11:22:33:44:55:66": "apple",
	}
	sortDevices(devices, SortLabel, true, tags)
	if devices[0].MAC != "11:22:33:44:55:66" {
		t.Errorf("sort by label asc: first MAC = %s, want 11:22:33:44:55:66", devices[0].MAC)
	}
}

func TestSortColumnName(t *testing.T) {
	tests := []struct {
		col  SortColumn
		want string
	}{
		{SortIP, "IP"},
		{SortMAC, "MAC"},
		{SortHostname, "Hostname"},
		{SortLabel, "Label"},
		{SortVendor, "Vendor"},
		{SortLastSeen, "Last Seen"},
	}
	for _, tt := range tests {
		if got := sortColumnName(tt.col); got != tt.want {
			t.Errorf("sortColumnName(%d) = %q, want %q", tt.col, got, tt.want)
		}
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
