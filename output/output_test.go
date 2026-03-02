package output

import (
	"bytes"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/fabioconcina/arpdvark/scanner"
	"github.com/fabioconcina/arpdvark/state"
)

var testTime = time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

func testDevices() []scanner.Device {
	return []scanner.Device{
		{
			IP:        net.ParseIP("192.168.1.1").To4(),
			MAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			Vendor:    "Cisco Systems",
			Hostname:  "router.local",
			FirstSeen: testTime,
			LastSeen:  testTime,
		},
		{
			IP:        net.ParseIP("192.168.1.10").To4(),
			MAC:       net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
			Vendor:    "Unknown",
			Hostname:  "",
			FirstSeen: testTime,
			LastSeen:  testTime,
		},
	}
}

func TestToDeviceJSON(t *testing.T) {
	tags := map[string]string{"aa:bb:cc:dd:ee:ff": "main-router"}
	result := ToDeviceJSON(testDevices(), tags)

	if len(result) != 2 {
		t.Fatalf("got %d devices, want 2", len(result))
	}

	d := result[0]
	if d.IP != "192.168.1.1" {
		t.Errorf("IP = %q, want %q", d.IP, "192.168.1.1")
	}
	if d.MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MAC = %q, want %q", d.MAC, "aa:bb:cc:dd:ee:ff")
	}
	if d.Vendor != "Cisco Systems" {
		t.Errorf("Vendor = %q, want %q", d.Vendor, "Cisco Systems")
	}
	if d.Hostname != "router.local" {
		t.Errorf("Hostname = %q, want %q", d.Hostname, "router.local")
	}
	if d.Label != "main-router" {
		t.Errorf("Label = %q, want %q", d.Label, "main-router")
	}
	if d.FirstSeen != "2025-01-15T10:30:00Z" {
		t.Errorf("FirstSeen = %q, want %q", d.FirstSeen, "2025-01-15T10:30:00Z")
	}

	// Second device has no label.
	if result[1].Label != "" {
		t.Errorf("Label = %q, want empty", result[1].Label)
	}
}

func TestToDeviceJSON_Empty(t *testing.T) {
	result := ToDeviceJSON(nil, nil)
	if len(result) != 0 {
		t.Errorf("got %d devices, want 0", len(result))
	}
}

func TestWriteJSON(t *testing.T) {
	tags := map[string]string{"aa:bb:cc:dd:ee:ff": "main-router"}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, testDevices(), tags); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	// Verify it round-trips.
	var parsed []DeviceJSON
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("got %d devices, want 2", len(parsed))
	}
	if parsed[0].IP != "192.168.1.1" {
		t.Errorf("parsed[0].IP = %q, want %q", parsed[0].IP, "192.168.1.1")
	}
	if parsed[0].Label != "main-router" {
		t.Errorf("parsed[0].Label = %q, want %q", parsed[0].Label, "main-router")
	}
}

func TestWriteTable(t *testing.T) {
	tags := map[string]string{"aa:bb:cc:dd:ee:ff": "main-router"}
	var buf bytes.Buffer
	WriteTable(&buf, testDevices(), tags)
	out := buf.String()

	// Header row must be present.
	if !strings.Contains(out, "IP") || !strings.Contains(out, "MAC") || !strings.Contains(out, "VENDOR") {
		t.Errorf("missing header columns in:\n%s", out)
	}

	// Data rows.
	if !strings.Contains(out, "192.168.1.1") {
		t.Errorf("missing IP 192.168.1.1 in:\n%s", out)
	}
	if !strings.Contains(out, "main-router") {
		t.Errorf("missing label in:\n%s", out)
	}
	if !strings.Contains(out, "Cisco Systems") {
		t.Errorf("missing vendor in:\n%s", out)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 { // header + 2 data rows
		t.Errorf("got %d lines, want 3", len(lines))
	}
}

func TestWriteTable_Empty(t *testing.T) {
	var buf bytes.Buffer
	WriteTable(&buf, nil, nil)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("got %d lines, want 1 (header only)", len(lines))
	}
}

func testStateDevices() []state.Device {
	return []state.Device{
		{
			MAC:       "aa:bb:cc:dd:ee:ff",
			IP:        "192.168.1.1",
			Vendor:    "Cisco Systems",
			Hostname:  "router.local",
			FirstSeen: testTime,
			LastSeen:  testTime,
			Online:    true,
		},
		{
			MAC:       "11:22:33:44:55:66",
			IP:        "192.168.1.10",
			Vendor:    "Unknown",
			Hostname:  "",
			FirstSeen: testTime,
			LastSeen:  testTime,
			Online:    false,
		},
	}
}

func TestToDeviceJSONFromState(t *testing.T) {
	tags := map[string]string{"aa:bb:cc:dd:ee:ff": "main-router"}
	result := ToDeviceJSONFromState(testStateDevices(), tags)

	if len(result) != 2 {
		t.Fatalf("got %d devices, want 2", len(result))
	}

	d := result[0]
	if d.IP != "192.168.1.1" {
		t.Errorf("IP = %q, want %q", d.IP, "192.168.1.1")
	}
	if d.Label != "main-router" {
		t.Errorf("Label = %q, want %q", d.Label, "main-router")
	}
	if d.Online == nil || *d.Online != true {
		t.Error("Online should be true")
	}

	if result[1].Online == nil || *result[1].Online != false {
		t.Error("second device Online should be false")
	}
}

func TestWriteJSONFromState(t *testing.T) {
	tags := map[string]string{"aa:bb:cc:dd:ee:ff": "main-router"}
	var buf bytes.Buffer
	if err := WriteJSONFromState(&buf, testStateDevices(), tags); err != nil {
		t.Fatalf("WriteJSONFromState error: %v", err)
	}

	var parsed []DeviceJSON
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("got %d devices, want 2", len(parsed))
	}
	if parsed[0].Online == nil || *parsed[0].Online != true {
		t.Error("first device should be online")
	}
	if parsed[1].Online == nil || *parsed[1].Online != false {
		t.Error("second device should be offline")
	}
}

func TestWriteJSON_OnlineOmitted(t *testing.T) {
	// Standard WriteJSON should not include the "online" field.
	var buf bytes.Buffer
	if err := WriteJSON(&buf, testDevices(), nil); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}
	if strings.Contains(buf.String(), "online") {
		t.Error("standard WriteJSON should not include 'online' field")
	}
}

func TestWriteTableFromState(t *testing.T) {
	tags := map[string]string{"aa:bb:cc:dd:ee:ff": "main-router"}
	var buf bytes.Buffer
	WriteTableFromState(&buf, testStateDevices(), tags)
	out := buf.String()

	if !strings.Contains(out, "STATUS") {
		t.Errorf("missing STATUS header in:\n%s", out)
	}
	if !strings.Contains(out, "online") {
		t.Errorf("missing 'online' status in:\n%s", out)
	}
	if !strings.Contains(out, "offline") {
		t.Errorf("missing 'offline' status in:\n%s", out)
	}
}
