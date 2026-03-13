package output

import (
	"encoding/json"
	"io"
	"time"

	"github.com/fabioconcina/arpdvark/scanner"
	"github.com/fabioconcina/arpdvark/state"
)

// DeviceJSON is the machine-readable representation of a discovered device.
type DeviceJSON struct {
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	Vendor    string `json:"vendor"`
	Hostname  string `json:"hostname"`
	Label     string `json:"label"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
	Online    *bool  `json:"online,omitempty"`
}

// ToDeviceJSON converts scanner devices and a tags map into a JSON-serializable slice.
func ToDeviceJSON(devices []scanner.Device, tags map[string]string) []DeviceJSON {
	out := make([]DeviceJSON, len(devices))
	for i, d := range devices {
		out[i] = DeviceJSON{
			IP:        d.IP.String(),
			MAC:       d.MAC.String(),
			Vendor:    d.Vendor,
			Hostname:  d.Hostname,
			Label:     tags[d.MAC.String()],
			FirstSeen: d.FirstSeen.Format(time.RFC3339),
			LastSeen:  d.LastSeen.Format(time.RFC3339),
		}
	}
	return out
}

// WriteJSON writes the device list as a pretty-printed JSON array to w.
func WriteJSON(w io.Writer, devices []scanner.Device, tags map[string]string) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(ToDeviceJSON(devices, tags))
}

// ToDeviceJSONFromState converts state devices with online status into a JSON-serializable slice.
func ToDeviceJSONFromState(devices []state.Device, tags map[string]string) []DeviceJSON {
	out := make([]DeviceJSON, len(devices))
	for i, d := range devices {
		online := d.Online
		out[i] = DeviceJSON{
			IP:        d.IP,
			MAC:       d.MAC,
			Vendor:    d.Vendor,
			Hostname:  d.Hostname,
			Label:     tags[d.MAC],
			FirstSeen: d.FirstSeen.Format(time.RFC3339),
			LastSeen:  d.LastSeen.Format(time.RFC3339),
			Online:    &online,
		}
	}
	return out
}

// WriteJSONFromState writes state devices as a pretty-printed JSON array to w.
func WriteJSONFromState(w io.Writer, devices []state.Device, tags map[string]string) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(ToDeviceJSONFromState(devices, tags))
}
