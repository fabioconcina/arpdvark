package tui

import (
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/fabioconcina/arpdvark/state"
)

const (
	colWidthIP       = 16
	colWidthMAC      = 21
	colWidthHostname = 28
	colWidthLabel    = 20
)

var (
	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	styleStatus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	styleScanning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true)

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	styleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true).
			Padding(0, 1)

	styleErr = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	styleDim = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	styleNew = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
)

// columnTitles are the base titles for each table column.
var columnTitles = [5]string{"IP Address", "MAC Address", "Hostname", "Label", "Vendor"}

// columnSortCol maps table column index to SortColumn.
var columnSortCol = [5]SortColumn{SortIP, SortMAC, SortHostname, SortLabel, SortVendor}

// sortIndicator returns the arrow string for the active sort column.
func sortIndicator(asc bool) string {
	if asc {
		return " ^"
	}
	return " v"
}

// buildColumns returns table columns with a sort indicator on the active column.
func buildColumns(sortCol SortColumn, sortAsc bool, vendorWidth int) []table.Column {
	widths := [5]int{colWidthIP, colWidthMAC, colWidthHostname, colWidthLabel, vendorWidth}
	cols := make([]table.Column, 5)
	for i := 0; i < 5; i++ {
		title := columnTitles[i]
		if columnSortCol[i] == sortCol {
			title += sortIndicator(sortAsc)
		}
		cols[i] = table.Column{Title: title, Width: widths[i]}
	}
	return cols
}

// newTable creates a configured bubbles table model with default dimensions.
func newTable() table.Model {
	cols := buildColumns(SortIP, true, 30)
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	s := table.DefaultStyles()
	s.Header = styleHeader
	s.Selected = styleSelected
	t.SetStyles(s)
	return t
}

// updateColumns updates column headers with the current sort indicator.
func updateColumns(m *M) {
	if m.width == 0 {
		return
	}
	vendorWidth := m.width - 2 - colWidthIP - colWidthMAC - colWidthHostname - colWidthLabel - 10
	if vendorWidth < 10 {
		vendorWidth = 10
	}
	m.tbl.SetColumns(buildColumns(m.sortCol, m.sortAsc, vendorWidth))
}

// applySize resizes the table to fill the given terminal dimensions.
// Each column cell has 1-char padding on each side (bubbles default Cell style),
// so total row width = sum(colWidth+2) for all columns.
func applySize(m M) M {
	if m.width == 0 || m.height == 0 {
		return m
	}
	// 2 chars for border (left + right), 10 for cell padding (5 cols × 2)
	vendorWidth := m.width - 2 - colWidthIP - colWidthMAC - colWidthHostname - colWidthLabel - 10
	if vendorWidth < 10 {
		vendorWidth = 10
	}
	m.tbl.SetColumns(buildColumns(m.sortCol, m.sortAsc, vendorWidth))
	// 2 border rows + 1 title + 1 status + 1 table header = 5 overhead rows
	tableHeight := m.height - 5
	if tableHeight < 1 {
		tableHeight = 1
	}
	m.tbl.SetHeight(tableHeight)
	m.tbl.SetRows(devicesToRows(m.filteredDevices(), m.tags, m.newMACs))
	return m
}

// devicesToRows converts state devices to table rows.
// Offline devices are rendered with dim styling.
// New devices (in newMACs) are rendered with green/bold styling.
// tags is a mac→label map; nil is treated as empty.
func devicesToRows(devices []state.Device, tags map[string]string, newMACs map[string]bool) []table.Row {
	rows := make([]table.Row, len(devices))
	for i, d := range devices {
		ip, mac, hostname, label, vendor := d.IP, d.MAC, d.Hostname, tags[d.MAC], d.Vendor
		if !d.Online {
			ip = styleDim.Render(ip)
			mac = styleDim.Render(mac)
			hostname = styleDim.Render(hostname)
			label = styleDim.Render(label)
			vendor = styleDim.Render(vendor)
		} else if newMACs[d.MAC] {
			ip = styleNew.Render(ip)
			mac = styleNew.Render(mac)
			hostname = styleNew.Render(hostname)
			label = styleNew.Render(label)
			vendor = styleNew.Render(vendor)
		}
		rows[i] = table.Row{ip, mac, hostname, label, vendor}
	}
	return rows
}

// sortColumnName returns the display name for a sort column.
func sortColumnName(col SortColumn) string {
	switch col {
	case SortIP:
		return "IP"
	case SortMAC:
		return "MAC"
	case SortHostname:
		return "Hostname"
	case SortLabel:
		return "Label"
	case SortVendor:
		return "Vendor"
	case SortLastSeen:
		return "Last Seen"
	default:
		return "IP"
	}
}

// sortDevices sorts a slice of devices in place by the given column and direction.
func sortDevices(devices []state.Device, col SortColumn, asc bool, tags map[string]string) {
	sort.SliceStable(devices, func(i, j int) bool {
		var less bool
		switch col {
		case SortIP:
			a := net.ParseIP(devices[i].IP)
			b := net.ParseIP(devices[j].IP)
			if a == nil || b == nil {
				less = devices[i].IP < devices[j].IP
			} else {
				ai := binary.BigEndian.Uint32(a.To4())
				bi := binary.BigEndian.Uint32(b.To4())
				less = ai < bi
			}
		case SortMAC:
			less = devices[i].MAC < devices[j].MAC
		case SortHostname:
			less = strings.ToLower(devices[i].Hostname) < strings.ToLower(devices[j].Hostname)
		case SortLabel:
			less = strings.ToLower(tags[devices[i].MAC]) < strings.ToLower(tags[devices[j].MAC])
		case SortVendor:
			less = strings.ToLower(devices[i].Vendor) < strings.ToLower(devices[j].Vendor)
		case SortLastSeen:
			less = devices[i].LastSeen.Before(devices[j].LastSeen)
		default:
			less = devices[i].IP < devices[j].IP
		}
		if !asc {
			return !less
		}
		return less
	})
}

// renderView builds the full TUI string from the model.
func renderView(m M) string {
	title := styleTitle.Render("arpdvark") +
		"  •  interface: " + m.iface +
		"  •  subnet: " + m.subnet

	tableStr := m.tbl.View()

	var statusLine string
	if m.editing {
		statusLine = styleStatus.Render("Label for "+m.editMAC+":") + " " + m.editInput.View()
	} else if m.filtering {
		statusLine = styleStatus.Render("/") + " " + m.filterInput.View()
	} else {
		var statusParts []string

		online := countOnline(m.allDevices)
		offline := len(m.allDevices) - online
		if offline > 0 {
			statusParts = append(statusParts, fmt.Sprintf("%d online, %d offline", online, offline))
		} else {
			statusParts = append(statusParts, fmt.Sprintf("%d device(s)", online))
		}

		if len(m.newMACs) > 0 {
			statusParts = append(statusParts, styleNew.Render(fmt.Sprintf("%d new", len(m.newMACs))))
		}

		if m.scanning {
			statusParts = append(statusParts, styleScanning.Render("scanning..."))
		} else if !m.lastScan.IsZero() {
			statusParts = append(statusParts, "last scan: "+humanDuration(time.Since(m.lastScan))+" ago")
		}

		if m.filterText != "" {
			statusParts = append(statusParts, fmt.Sprintf("filter: %q", m.filterText))
		}

		dir := "asc"
		if !m.sortAsc {
			dir = "desc"
		}
		if m.sortCol != SortIP || !m.sortAsc {
			statusParts = append(statusParts, fmt.Sprintf("sort: %s %s", sortColumnName(m.sortCol), dir))
		}

		if m.err != nil {
			statusParts = append(statusParts, styleErr.Render("error: "+m.err.Error()))
		}

		statusParts = append(statusParts, "↵: details  e: label  ←→: sort col  s: sort dir  /: filter  o: offline  r: rescan  q: quit")
		statusLine = styleStatus.Render(strings.Join(statusParts, "  •  "))
	}

	body := strings.Join([]string{title, tableStr, statusLine}, "\n")
	bs := styleBorder
	if m.width > 0 {
		bs = bs.Width(m.width - 2)
	}
	return bs.Render(body)
}

// detailFieldCount is the number of fields shown in the detail view.
const detailFieldCount = 8

// renderDetail renders the device detail panel.
func renderDetail(m M) string {
	d := m.detailDevice
	title := styleTitle.Render("device detail") + "  —  " + d.IP

	label := m.tags[d.MAC]
	if label == "" {
		label = "-"
	}
	status := "offline"
	if d.Online {
		status = "online"
	}

	fields := [detailFieldCount][2]string{
		{"IP Address", d.IP},
		{"MAC Address", d.MAC},
		{"Hostname", d.Hostname},
		{"Label", label},
		{"Vendor", d.Vendor},
		{"Status", status},
		{"First Seen", d.FirstSeen.Format("2006-01-02 15:04")},
		{"Last Seen", d.LastSeen.Format("2006-01-02 15:04")},
	}

	styleFieldName := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Width(12)
	styleFieldValue := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	var rows []string
	for i, f := range fields {
		row := styleFieldName.Render(f[0]) + "  " + styleFieldValue.Render(f[1])
		if i == m.detailCursor {
			row = styleSelected.Render(fmt.Sprintf("%-12s  %s", f[0], f[1]))
		}
		rows = append(rows, row)
	}

	statusLine := styleStatus.Render("↑↓: navigate  esc / ↵: back")

	body := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + statusLine
	bs := styleBorder
	if m.width > 0 {
		bs = bs.Width(m.width - 2)
	}
	return bs.Render(body)
}

// renderSplash renders the logo centered in the terminal for the splash screen.
func renderSplash(m M) string {
	g := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	d := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	logo := g.Render(
		"▄▀█ █▀█ █▀█ █▀▄ █ █ ▄▀█ █▀█ █▄▀\n" +
			"█▀█ ██▀ █▀▀ █ █ ▀▄▀ █▀█ ██▀ ██▄\n" +
			"▀ ▀ ▀ ▀ ▀   ▀▀  ▀ ▀ ▀ ▀ ▀ ▀ ▀ ▀",
	)
	sep := d.Render("─────────────────────────────────")
	sub := "ARP network scanner  ·  " + m.version

	content := logo + "\n" + sep + "\n" + sub
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}

// countOnline returns the number of online devices in the list.
func countOnline(devices []state.Device) int {
	n := 0
	for _, d := range devices {
		if d.Online {
			n++
		}
	}
	return n
}

// humanDuration formats a duration as a short human-readable string.
func humanDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
