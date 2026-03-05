package activity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordIncrements(t *testing.T) {
	s := Empty()
	// Wednesday 14:00
	ts := time.Date(2025, 3, 5, 14, 30, 0, 0, time.UTC)
	s.Record([]string{"aa:bb:cc:dd:ee:ff"}, ts)
	s.Record([]string{"aa:bb:cc:dd:ee:ff"}, ts)

	m := s.Get("aa:bb:cc:dd:ee:ff")
	if m == nil {
		t.Fatal("expected non-nil matrix")
	}
	if m[ts.Weekday()][14] != 2 {
		t.Errorf("expected 2, got %d", m[ts.Weekday()][14])
	}
}

func TestRecordMultipleMACs(t *testing.T) {
	s := Empty()
	ts := time.Date(2025, 3, 5, 10, 0, 0, 0, time.UTC)
	s.Record([]string{"aa:aa:aa:aa:aa:aa", "bb:bb:bb:bb:bb:bb"}, ts)

	for _, mac := range []string{"aa:aa:aa:aa:aa:aa", "bb:bb:bb:bb:bb:bb"} {
		m := s.Get(mac)
		if m == nil {
			t.Fatalf("expected non-nil matrix for %s", mac)
		}
		if m[ts.Weekday()][10] != 1 {
			t.Errorf("%s: expected 1, got %d", mac, m[ts.Weekday()][10])
		}
	}
}

func TestGetNil(t *testing.T) {
	s := Empty()
	if m := s.Get("unknown"); m != nil {
		t.Error("expected nil for unknown MAC")
	}
}

func TestGetReturnsCopy(t *testing.T) {
	s := Empty()
	ts := time.Date(2025, 1, 6, 8, 0, 0, 0, time.UTC) // Monday
	s.Record([]string{"aa:bb:cc:dd:ee:ff"}, ts)

	m := s.Get("aa:bb:cc:dd:ee:ff")
	m[time.Monday][8] = 999

	m2 := s.Get("aa:bb:cc:dd:ee:ff")
	if m2[time.Monday][8] == 999 {
		t.Error("Get should return a copy, not a reference")
	}
}

func TestForget(t *testing.T) {
	s := Empty()
	ts := time.Date(2025, 3, 5, 12, 0, 0, 0, time.UTC)
	s.Record([]string{"aa:bb:cc:dd:ee:ff"}, ts)
	s.Forget("aa:bb:cc:dd:ee:ff")
	if m := s.Get("aa:bb:cc:dd:ee:ff"); m != nil {
		t.Error("expected nil after Forget")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.json")
	s := &Store{data: make(map[string]*Matrix), path: path}

	ts := time.Date(2025, 3, 5, 14, 0, 0, 0, time.UTC)
	s.Record([]string{"aa:bb:cc:dd:ee:ff"}, ts)
	s.Record([]string{"aa:bb:cc:dd:ee:ff"}, ts)

	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload from disk.
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()
	loaded := make(map[string]*Matrix)
	if err := json.NewDecoder(f).Decode(&loaded); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	m := loaded["aa:bb:cc:dd:ee:ff"]
	if m == nil {
		t.Fatal("expected non-nil matrix after reload")
	}
	if m[ts.Weekday()][14] != 2 {
		t.Errorf("expected 2 after reload, got %d", m[ts.Weekday()][14])
	}
}

func TestHeatmapNil(t *testing.T) {
	result := Heatmap(nil)
	if result != "No activity data yet" {
		t.Errorf("unexpected: %q", result)
	}
}

func TestHeatmapEmpty(t *testing.T) {
	m := &Matrix{}
	result := Heatmap(m)
	if result != "No activity data yet" {
		t.Errorf("unexpected: %q", result)
	}
}

func TestHeatmapSingleCell(t *testing.T) {
	m := &Matrix{}
	m[time.Monday][12] = 5 // Only nonzero cell → should render as max intensity

	result := Heatmap(m)
	lines := strings.Split(result, "\n")
	if len(lines) != 8 { // 1 header + 7 days
		t.Fatalf("expected 8 lines, got %d", len(lines))
	}
	// Monday is first day row (index 1), hour 12 is character at offset 5+12=17
	monLine := lines[1]
	runes := []rune(monLine)
	if len(runes) < 18 {
		t.Fatalf("Monday line too short: %q", monLine)
	}
	// The block at hour 12 should be '█' (max)
	if runes[5+12] != '█' {
		t.Errorf("expected max block at hour 12, got %c", runes[5+12])
	}
}

func TestHeatmapNormalization(t *testing.T) {
	m := &Matrix{}
	m[time.Monday][0] = 100  // max → level 4 (█)
	m[time.Monday][1] = 80   // 0.8 → level 4 (█)
	m[time.Monday][2] = 60   // 0.6 → level 3 (▓)
	m[time.Monday][3] = 30   // 0.3 → level 2 (▒)
	m[time.Monday][4] = 10   // 0.1 → level 1 (░)
	m[time.Monday][5] = 0    // 0   → level 0 ( )

	result := Heatmap(m)
	lines := strings.Split(result, "\n")
	monRunes := []rune(lines[1])

	expected := []rune{'█', '█', '▓', '▒', '░', ' '}
	for i, exp := range expected {
		got := monRunes[5+i]
		if got != exp {
			t.Errorf("hour %d: expected %c, got %c", i, exp, got)
		}
	}
}
