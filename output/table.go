package output

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/fabioconcina/arpdvark/scanner"
	"github.com/fabioconcina/arpdvark/state"
)

// fmtLatency formats a duration as a short latency string for plain-text output.
func fmtLatency(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	ms := float64(d) / float64(time.Millisecond)
	if ms < 10 {
		return fmt.Sprintf("%.1fms", ms)
	}
	return fmt.Sprintf("%dms", int(ms))
}

// WriteTable writes a human-readable tab-aligned table to w.
// Output contains no ANSI escape codes and is suitable for piping to grep/awk/cut.
func WriteTable(w io.Writer, devices []scanner.Device, tags map[string]string) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "IP\tMAC\tHOSTNAME\tLABEL\tVENDOR\tLATENCY")
	for _, d := range devices {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			d.IP, d.MAC, d.Hostname, tags[d.MAC.String()], d.Vendor, fmtLatency(d.Latency))
	}
	tw.Flush()
}

// WriteTableFromState writes a table that includes a STATUS column for online/offline devices.
func WriteTableFromState(w io.Writer, devices []state.Device, tags map[string]string) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "IP\tMAC\tHOSTNAME\tLABEL\tVENDOR\tLATENCY\tSTATUS")
	for _, d := range devices {
		status := "online"
		if !d.Online {
			status = "offline"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			d.IP, d.MAC, d.Hostname, tags[d.MAC], d.Vendor, fmtLatency(d.Latency), status)
	}
	tw.Flush()
}
