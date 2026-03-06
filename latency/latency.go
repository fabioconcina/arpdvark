package latency

import (
	"encoding/json"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const bufSize = 100

// History holds a circular buffer of latency samples for one device.
type History struct {
	Samples [bufSize]float64 `json:"samples"` // milliseconds, 0 = unused
	Next    int              `json:"next"`     // write index
	Count   int              `json:"count"`    // total recorded (capped at bufSize)
}

// Stats holds IQR summary statistics (all values in milliseconds).
type Stats struct {
	Min    float64
	Q1     float64
	Median float64
	Q3     float64
	Max    float64
	Count  int
}

// Store holds per-MAC latency histories in memory and can persist to disk.
type Store struct {
	mu   sync.RWMutex
	data map[string]*History
	path string
}

func configDir() (string, error) {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err == nil {
			return filepath.Join(u.HomeDir, ".config"), nil
		}
	}
	return os.UserConfigDir()
}

// Load reads ~/.config/arpdvark/latency.json.
// Returns an empty Store if the file does not exist yet.
func Load() (*Store, error) {
	base, err := configDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(base, "arpdvark", "latency.json")
	s := &Store{data: make(map[string]*History), path: path}
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
	return &Store{data: make(map[string]*History)}
}

// RecordAll records latency measurements for a batch of devices.
func (s *Store) RecordAll(latencies map[string]time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for mac, d := range latencies {
		h := s.data[mac]
		if h == nil {
			h = &History{}
			s.data[mac] = h
		}
		ms := float64(d) / float64(time.Millisecond)
		h.Samples[h.Next] = ms
		h.Next = (h.Next + 1) % bufSize
		if h.Count < bufSize {
			h.Count++
		}
	}
}

// Get returns computed IQR stats for the given MAC, or nil if no data.
func (s *Store) Get(mac string) *Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := s.data[mac]
	if h == nil || h.Count == 0 {
		return nil
	}
	return computeStats(h)
}

// Forget removes latency data for the given MAC.
func (s *Store) Forget(mac string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, mac)
}

// Save persists latency data to disk atomically.
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

// computeStats calculates the five-number summary from a History.
func computeStats(h *History) *Stats {
	n := h.Count
	vals := make([]float64, n)
	// Extract valid samples from circular buffer.
	if n < bufSize {
		copy(vals, h.Samples[:n])
	} else {
		// Buffer is full; samples start at h.Next (oldest).
		copy(vals, h.Samples[h.Next:])
		copy(vals[bufSize-h.Next:], h.Samples[:h.Next])
	}
	sort.Float64s(vals)

	st := &Stats{
		Min:    vals[0],
		Max:    vals[n-1],
		Median: median(vals),
		Count:  n,
	}

	if n >= 4 {
		mid := n / 2
		if n%2 == 0 {
			st.Q1 = median(vals[:mid])
			st.Q3 = median(vals[mid:])
		} else {
			st.Q1 = median(vals[:mid])
			st.Q3 = median(vals[mid+1:])
		}
	} else {
		st.Q1 = st.Min
		st.Q3 = st.Max
	}

	// Round to 2 decimal places.
	st.Min = math.Round(st.Min*100) / 100
	st.Q1 = math.Round(st.Q1*100) / 100
	st.Median = math.Round(st.Median*100) / 100
	st.Q3 = math.Round(st.Q3*100) / 100
	st.Max = math.Round(st.Max*100) / 100

	return st
}

// median returns the median of a sorted slice.
func median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}
