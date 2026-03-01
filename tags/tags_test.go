package tags

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetSet(t *testing.T) {
	s := &Store{data: make(map[string]string), path: filepath.Join(t.TempDir(), "tags.json")}

	if got := s.Get("aa:bb:cc:dd:ee:ff"); got != "" {
		t.Errorf("Get on empty store = %q, want \"\"", got)
	}

	if err := s.Set("aa:bb:cc:dd:ee:ff", "my-router"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := s.Get("aa:bb:cc:dd:ee:ff"); got != "my-router" {
		t.Errorf("Get = %q, want %q", got, "my-router")
	}
}

func TestSetEmptyDeletesKey(t *testing.T) {
	s := &Store{data: make(map[string]string), path: filepath.Join(t.TempDir(), "tags.json")}
	if err := s.Set("aa:bb:cc:dd:ee:ff", "router"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Set("aa:bb:cc:dd:ee:ff", ""); err != nil {
		t.Fatalf("Set empty: %v", err)
	}
	if got := s.Get("aa:bb:cc:dd:ee:ff"); got != "" {
		t.Errorf("Get after delete = %q, want \"\"", got)
	}
	// Key must not appear in the saved JSON.
	data, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := m["aa:bb:cc:dd:ee:ff"]; ok {
		t.Errorf("deleted key still present in JSON")
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tags.json")
	s1 := &Store{data: make(map[string]string), path: path}
	if err := s1.Set("11:22:33:44:55:66", "NAS"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Reload from same file.
	s2 := &Store{data: make(map[string]string), path: path}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&s2.data); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got := s2.Get("11:22:33:44:55:66"); got != "NAS" {
		t.Errorf("after reload got %q, want %q", got, "NAS")
	}
}

func TestAll(t *testing.T) {
	s := &Store{data: map[string]string{"a": "x", "b": "y"}}
	all := s.All()
	if len(all) != 2 {
		t.Errorf("All() len = %d, want 2", len(all))
	}
	// Mutation of returned map must not affect Store.
	all["c"] = "z"
	if _, ok := s.data["c"]; ok {
		t.Errorf("All() returned non-copy")
	}
}

func TestEmptyStore(t *testing.T) {
	s := Empty()
	// Set on an empty (pathless) store succeeds silently.
	if err := s.Set("aa:bb:cc:dd:ee:ff", "test"); err != nil {
		t.Errorf("Set on Empty store returned error: %v", err)
	}
	if got := s.Get("aa:bb:cc:dd:ee:ff"); got != "test" {
		t.Errorf("Get = %q, want %q", got, "test")
	}
}
