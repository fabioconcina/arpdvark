package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/fabioconcina/arpdvark/scanner"
)

// WriteTable writes a human-readable tab-aligned table to w.
// Output contains no ANSI escape codes and is suitable for piping to grep/awk/cut.
func WriteTable(w io.Writer, devices []scanner.Device, tags map[string]string) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "IP\tMAC\tHOSTNAME\tLABEL\tVENDOR")
	for _, d := range devices {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			d.IP, d.MAC, d.Hostname, tags[d.MAC.String()], d.Vendor)
	}
	tw.Flush()
}
