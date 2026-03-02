package state

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fabioconcina/arpdvark/scanner"
)

// Record is a single persisted device entry, keyed by MAC address.
type Record struct {
	IP        string    `json:"ip"`
	Vendor    string    `json:"vendor"`
	Hostname  string    `json:"hostname"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// Device is a Record enriched with its MAC and online status, used for display.
type Device struct {
	MAC       string
	IP        string
	Vendor    string
	Hostname  string
	FirstSeen time.Time
	LastSeen  time.Time
	Online    bool
}

// Store holds persisted device records and manages the state file.
type Store struct {
	mu   sync.RWMutex
	data map[string]Record // keyed by MAC string
	path string
}

// configDir returns the config base directory, respecting $SUDO_USER so that
// state is stored in the invoking user's home rather than root's.
func configDir() (string, error) {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err == nil {
			return filepath.Join(u.HomeDir, ".config"), nil
		}
	}
	return os.UserConfigDir()
}

// Load reads ~/.config/arpdvark/state.json (creates parent dirs if absent).
// Returns an empty Store (not an error) if the file does not exist yet.
func Load() (*Store, error) {
	base, err := configDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(base, "arpdvark", "state.json")
	s := &Store{data: make(map[string]Record), path: path}
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
	return &Store{data: make(map[string]Record)}
}

// Merge integrates scan results into the persisted state.
// For each scanned device: update IP/vendor/hostname/LastSeen; preserve FirstSeen.
// Returns all known devices with online status set. Persists to disk after merge.
func (s *Store) Merge(scanned []scanner.Device) ([]Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	onlineMACs := make(map[string]bool, len(scanned))
	for _, d := range scanned {
		mac := d.MAC.String()
		onlineMACs[mac] = true

		existing, ok := s.data[mac]
		if ok {
			existing.IP = d.IP.String()
			existing.Vendor = d.Vendor
			existing.Hostname = d.Hostname
			existing.LastSeen = d.LastSeen
		} else {
			existing = Record{
				IP:        d.IP.String(),
				Vendor:    d.Vendor,
				Hostname:  d.Hostname,
				FirstSeen: d.FirstSeen,
				LastSeen:  d.LastSeen,
			}
		}
		s.data[mac] = existing
	}

	if err := s.save(); err != nil {
		return nil, err
	}

	return s.allDevices(onlineMACs), nil
}

// All returns all known devices, all marked offline (no scan context).
// Useful for pre-scan display of previously known devices.
func (s *Store) All() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.allDevices(nil)
}

// Forget removes a device by MAC address and persists to disk.
func (s *Store) Forget(mac string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, mac)
	return s.save()
}

// ForgetOlderThan removes devices whose LastSeen is before cutoff.
// Returns the number of removed entries.
func (s *Store) ForgetOlderThan(cutoff time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for mac, r := range s.data {
		if r.LastSeen.Before(cutoff) {
			delete(s.data, mac)
			count++
		}
	}
	if count > 0 {
		if err := s.save(); err != nil {
			return 0, err
		}
	}
	return count, nil
}

// allDevices returns a sorted slice of all devices with online status set.
// Caller must hold s.mu.
func (s *Store) allDevices(onlineMACs map[string]bool) []Device {
	out := make([]Device, 0, len(s.data))
	for mac, r := range s.data {
		out = append(out, Device{
			MAC:       mac,
			IP:        r.IP,
			Vendor:    r.Vendor,
			Hostname:  r.Hostname,
			FirstSeen: r.FirstSeen,
			LastSeen:  r.LastSeen,
			Online:    onlineMACs[mac],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		a := net.ParseIP(out[i].IP)
		b := net.ParseIP(out[j].IP)
		if a == nil || b == nil {
			return out[i].IP < out[j].IP
		}
		ai := binary.BigEndian.Uint32(a.To4())
		bi := binary.BigEndian.Uint32(b.To4())
		return ai < bi
	})
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
