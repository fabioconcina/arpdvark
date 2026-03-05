package activity

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"time"
)

// Matrix is a 7×24 grid of sighting counts: [weekday][hour].
// Weekday indices follow time.Weekday (Sunday=0 … Saturday=6).
type Matrix [7][24]uint32

// Store holds per-MAC activity matrices in memory and can persist to disk.
type Store struct {
	mu   sync.RWMutex
	data map[string]*Matrix // keyed by MAC string
	path string
}

// configDir returns the config base directory, respecting $SUDO_USER so that
// data is stored in the invoking user's home rather than root's.
func configDir() (string, error) {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err == nil {
			return filepath.Join(u.HomeDir, ".config"), nil
		}
	}
	return os.UserConfigDir()
}

// Load reads ~/.config/arpdvark/activity.json.
// Returns an empty Store if the file does not exist yet.
func Load() (*Store, error) {
	base, err := configDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(base, "arpdvark", "activity.json")
	s := &Store{data: make(map[string]*Matrix), path: path}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&s.data); err != nil {
		return nil, err
	}
	return s, nil
}

// Empty returns an in-memory-only Store (no file path).
func Empty() *Store {
	return &Store{data: make(map[string]*Matrix)}
}

// Record increments the activity count for the given MACs at time t.
// Updates are in-memory only; call Save to persist.
func (s *Store) Record(macs []string, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wd := t.Weekday()
	hr := t.Hour()
	for _, mac := range macs {
		m := s.data[mac]
		if m == nil {
			m = &Matrix{}
			s.data[mac] = m
		}
		m[wd][hr]++
	}
}

// Get returns a copy of the activity matrix for the given MAC, or nil if none.
func (s *Store) Get(mac string) *Matrix {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.data[mac]
	if m == nil {
		return nil
	}
	cp := *m
	return &cp
}

// Forget removes activity data for the given MAC.
func (s *Store) Forget(mac string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, mac)
}

// Save persists activity data to disk atomically.
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, s.path)
}
