package vendor_db

import (
	_ "embed"
	"fmt"
	"net"
	"strings"
	"sync"
)

//go:embed oui.txt
var ouiData string

var (
	db   map[string]string
	once sync.Once
)

func init() {
	once.Do(parseOUI)
}

// parseOUI reads the embedded oui.txt and builds the lookup map.
// Each line looks like:  A4-C3-F0   (hex)		Apple, Inc.
func parseOUI() {
	db = make(map[string]string, 30000)
	for _, line := range strings.Split(ouiData, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "(hex)") {
			continue
		}
		parts := strings.SplitN(line, "(hex)", 2)
		if len(parts) != 2 {
			continue
		}
		prefix := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(parts[0]), "-", ""))
		vendor := strings.TrimSpace(parts[1])
		if len(prefix) == 6 && vendor != "" {
			db[prefix] = vendor
		}
	}
}

// Lookup returns the vendor name for the given MAC address, or "Unknown".
// Locally administered MACs (bit 1 of first octet set) are flagged explicitly
// since they have no OUI registration — they are randomized, VM-assigned, or manually set.
func Lookup(mac net.HardwareAddr) string {
	if len(mac) < 3 {
		return "Unknown"
	}
	if mac[0]&0x02 != 0 {
		return "Local/Randomized"
	}
	key := fmt.Sprintf("%02X%02X%02X", mac[0], mac[1], mac[2])
	if v, ok := db[key]; ok {
		return v
	}
	return "Unknown"
}
