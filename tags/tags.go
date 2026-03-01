package tags

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"sync"
)

// Store holds a MAC→label map and persists it as JSON on disk.
type Store struct {
	mu   sync.RWMutex
	data map[string]string
	path string
}

// configDir returns the config base directory, respecting $SUDO_USER so that
// tags are stored in the invoking user's home rather than root's.
func configDir() (string, error) {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err == nil {
			return filepath.Join(u.HomeDir, ".config"), nil
		}
	}
	return os.UserConfigDir()
}

// Load reads ~/.config/arpdvark/tags.json (creates parent dirs if absent).
// Returns an empty Store (not an error) if the file does not exist yet.
func Load() (*Store, error) {
	base, err := configDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(base, "arpdvark", "tags.json")
	s := &Store{data: make(map[string]string), path: path}
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

// Empty returns an in-memory-only Store (no file path). Set() will return
// an error on save attempts, but Get() and All() work normally.
func Empty() *Store {
	return &Store{data: make(map[string]string)}
}

// Get returns the label for mac, or "" if unset.
func (s *Store) Get(mac string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[mac]
}

// Set updates the label for mac and persists to disk immediately.
// An empty label removes the key.
func (s *Store) Set(mac, label string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if label == "" {
		delete(s.data, mac)
	} else {
		s.data[mac] = label
	}
	return s.save()
}

// All returns a snapshot copy of all mac→label entries.
func (s *Store) All() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}

// save writes data to disk atomically (write tmp, then rename).
// Caller must hold s.mu (write lock).
func (s *Store) save() error {
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
