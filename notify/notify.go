package notify

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fabioconcina/arpdvark/scanner"
)

// Send posts a plain-text notification listing new devices to the given URL.
// Returns nil immediately if url is empty or devices is empty.
func Send(url string, devices []scanner.Device) error {
	if url == "" || len(devices) == 0 {
		return nil
	}

	var lines []string
	for _, d := range devices {
		line := fmt.Sprintf("New device: %s (%s)", d.IP, d.MAC)
		if d.Vendor != "" {
			line += " " + d.Vendor
		}
		if d.Hostname != "" {
			line += " " + d.Hostname
		}
		lines = append(lines, line)
	}
	body := strings.Join(lines, "\n")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "text/plain", strings.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("notify: server returned %s", resp.Status)
	}
	return nil
}
