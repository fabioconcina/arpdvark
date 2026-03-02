package state

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fabioconcina/arpdvark/scanner"
)

func makeDevice(ip, mac string) scanner.Device {
	hwMAC, _ := net.ParseMAC(mac)
	return scanner.Device{
		IP:        net.ParseIP(ip),
		MAC:       hwMAC,
		Vendor:    "TestVendor",
		Hostname:  "test.local",
		FirstSeen: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		LastSeen:  time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	}
}

func TestMergeNewDevices(t *testing.T) {
	s := &Store{data: make(map[string]Record), path: filepath.Join(t.TempDir(), "state.json")}
	devices := []scanner.Device{
		makeDevice("192.168.1.1", "aa:bb:cc:dd:ee:ff"),
		makeDevice("192.168.1.2", "11:22:33:44:55:66"),
	}

	all, err := s.Merge(devices)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("Merge returned %d devices, want 2", len(all))
	}
	for _, d := range all {
		if !d.Online {
			t.Errorf("device %s should be online", d.MAC)
		}
	}
}

func TestMergeUpdateExisting(t *testing.T) {
	s := &Store{data: make(map[string]Record), path: filepath.Join(t.TempDir(), "state.json")}

	// First merge.
	first := makeDevice("192.168.1.1", "aa:bb:cc:dd:ee:ff")
	first.FirstSeen = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	first.LastSeen = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := s.Merge([]scanner.Device{first}); err != nil {
		t.Fatalf("Merge 1: %v", err)
	}

	// Second merge with updated LastSeen and new IP.
	second := makeDevice("192.168.1.50", "aa:bb:cc:dd:ee:ff")
	second.FirstSeen = time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC) // should be ignored
	second.LastSeen = time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	all, err := s.Merge([]scanner.Device{second})
	if err != nil {
		t.Fatalf("Merge 2: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("got %d devices, want 1", len(all))
	}
	d := all[0]
	// FirstSeen should be preserved from first merge.
	if !d.FirstSeen.Equal(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("FirstSeen = %v, want 2025-01-01", d.FirstSeen)
	}
	// LastSeen should be updated.
	if !d.LastSeen.Equal(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("LastSeen = %v, want 2025-02-01", d.LastSeen)
	}
	// IP should be updated.
	if d.IP != "192.168.1.50" {
		t.Errorf("IP = %s, want 192.168.1.50", d.IP)
	}
}

func TestMergeOnlineStatus(t *testing.T) {
	s := &Store{data: make(map[string]Record), path: filepath.Join(t.TempDir(), "state.json")}

	// Merge two devices.
	devices := []scanner.Device{
		makeDevice("192.168.1.1", "aa:bb:cc:dd:ee:ff"),
		makeDevice("192.168.1.2", "11:22:33:44:55:66"),
	}
	if _, err := s.Merge(devices); err != nil {
		t.Fatalf("Merge 1: %v", err)
	}

	// Second merge with only one device — the other should be offline.
	all, err := s.Merge([]scanner.Device{devices[0]})
	if err != nil {
		t.Fatalf("Merge 2: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("got %d devices, want 2", len(all))
	}

	online := 0
	offline := 0
	for _, d := range all {
		if d.Online {
			online++
		} else {
			offline++
		}
	}
	if online != 1 || offline != 1 {
		t.Errorf("online=%d offline=%d, want 1 and 1", online, offline)
	}
}

func TestAll(t *testing.T) {
	s := &Store{data: map[string]Record{
		"aa:bb:cc:dd:ee:ff": {IP: "192.168.1.1", LastSeen: time.Now()},
	}}

	all := s.All()
	if len(all) != 1 {
		t.Fatalf("All() len = %d, want 1", len(all))
	}
	if all[0].Online {
		t.Error("All() should return devices as offline")
	}
}

func TestForget(t *testing.T) {
	s := &Store{data: make(map[string]Record), path: filepath.Join(t.TempDir(), "state.json")}
	devices := []scanner.Device{
		makeDevice("192.168.1.1", "aa:bb:cc:dd:ee:ff"),
		makeDevice("192.168.1.2", "11:22:33:44:55:66"),
	}
	if _, err := s.Merge(devices); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if err := s.Forget("aa:bb:cc:dd:ee:ff"); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	all := s.All()
	if len(all) != 1 {
		t.Fatalf("after Forget, got %d devices, want 1", len(all))
	}
	if all[0].MAC != "11:22:33:44:55:66" {
		t.Errorf("remaining device MAC = %s, want 11:22:33:44:55:66", all[0].MAC)
	}
}

func TestForgetOlderThan(t *testing.T) {
	s := &Store{data: make(map[string]Record), path: filepath.Join(t.TempDir(), "state.json")}
	old := makeDevice("192.168.1.1", "aa:bb:cc:dd:ee:ff")
	old.LastSeen = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := makeDevice("192.168.1.2", "11:22:33:44:55:66")
	recent.LastSeen = time.Now()

	if _, err := s.Merge([]scanner.Device{old, recent}); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	cutoff := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	n, err := s.ForgetOlderThan(cutoff)
	if err != nil {
		t.Fatalf("ForgetOlderThan: %v", err)
	}
	if n != 1 {
		t.Errorf("removed %d, want 1", n)
	}

	all := s.All()
	if len(all) != 1 {
		t.Fatalf("after ForgetOlderThan, got %d devices, want 1", len(all))
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s1 := &Store{data: make(map[string]Record), path: path}

	devices := []scanner.Device{makeDevice("192.168.1.1", "aa:bb:cc:dd:ee:ff")}
	if _, err := s1.Merge(devices); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	// Reload from same file.
	s2 := &Store{data: make(map[string]Record), path: path}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&s2.data); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	all := s2.All()
	if len(all) != 1 {
		t.Fatalf("after reload, got %d devices, want 1", len(all))
	}
	if all[0].IP != "192.168.1.1" {
		t.Errorf("IP = %s, want 192.168.1.1", all[0].IP)
	}
}

func TestEmptyStore(t *testing.T) {
	s := Empty()
	devices := []scanner.Device{makeDevice("192.168.1.1", "aa:bb:cc:dd:ee:ff")}
	all, err := s.Merge(devices)
	if err != nil {
		t.Fatalf("Merge on Empty: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("got %d devices, want 1", len(all))
	}
}
