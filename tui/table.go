package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/fabioconcina/arpdvark/scanner"
)

const (
	colWidthIP       = 16
	colWidthMAC      = 21
	colWidthHostname = 28
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
			Background(lipgloss.Color("235"))

	styleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)

	styleErr = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

// newTable creates a configured bubbles table model with default dimensions.
func newTable() table.Model {
	cols := []table.Column{
		{Title: "IP Address", Width: colWidthIP},
		{Title: "MAC Address", Width: colWidthMAC},
		{Title: "Hostname", Width: colWidthHostname},
		{Title: "Vendor", Width: 30},
	}
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

// applySize resizes the table to fill the given terminal dimensions.
// Each column cell has 1-char padding on each side (bubbles default Cell style),
// so total row width = sum(colWidth+2) for all columns.
func applySize(m M) M {
	if m.width == 0 || m.height == 0 {
		return m
	}
	// 2 chars for border (left + right), 8 for cell padding (4 cols × 2)
	vendorWidth := m.width - 2 - colWidthIP - colWidthMAC - colWidthHostname - 8
	if vendorWidth < 10 {
		vendorWidth = 10
	}
	m.tbl.SetColumns([]table.Column{
		{Title: "IP Address", Width: colWidthIP},
		{Title: "MAC Address", Width: colWidthMAC},
		{Title: "Hostname", Width: colWidthHostname},
		{Title: "Vendor", Width: vendorWidth},
	})
	// 2 border rows + 1 title + 1 status + 1 table header = 5 overhead rows
	tableHeight := m.height - 5
	if tableHeight < 1 {
		tableHeight = 1
	}
	m.tbl.SetHeight(tableHeight)
	m.tbl.SetRows(devicesToRows(m.devices))
	return m
}

// devicesToRows converts scanner devices to table rows.
func devicesToRows(devices []scanner.Device) []table.Row {
	rows := make([]table.Row, len(devices))
	for i, d := range devices {
		rows[i] = table.Row{d.IP.String(), d.MAC.String(), d.Hostname, d.Vendor}
	}
	return rows
}

// renderView builds the full TUI string from the model.
func renderView(m M) string {
	title := styleTitle.Render("arpdvark") +
		"  •  interface: " + m.iface +
		"  •  subnet: " + m.subnet

	tableStr := m.tbl.View()

	var statusParts []string
	statusParts = append(statusParts, fmt.Sprintf("%d device(s)", len(m.devices)))

	if m.scanning {
		statusParts = append(statusParts, styleScanning.Render("scanning…"))
	} else if !m.lastScan.IsZero() {
		statusParts = append(statusParts, "last scan: "+humanDuration(time.Since(m.lastScan))+" ago")
	}

	if m.err != nil {
		statusParts = append(statusParts, styleErr.Render("error: "+m.err.Error()))
	}

	statusParts = append(statusParts, "r: rescan  q: quit")
	status := styleStatus.Render(strings.Join(statusParts, "  •  "))

	body := strings.Join([]string{title, tableStr, status}, "\n")
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
