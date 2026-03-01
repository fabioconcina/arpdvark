package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"arpdvark/scanner"
)

// ScanCompleteMsg is sent when a scan finishes (successfully or not).
type ScanCompleteMsg struct {
	Devices []scanner.Device
	Err     error
}

// TickMsg is sent by the refresh timer to trigger the next scan.
type TickMsg time.Time

// M is the Bubbletea model for arpdvark.
type M struct {
	tbl      table.Model
	devices  []scanner.Device
	iface    string
	subnet   string
	lastScan time.Time
	scanning bool
	interval time.Duration
	err      error
	sc       *scanner.Scanner
	width    int
	height   int
}

// New creates a new TUI model.
func New(sc *scanner.Scanner, interval time.Duration) M {
	return M{
		tbl:      newTable(),
		iface:    sc.Interface(),
		subnet:   sc.Subnet(),
		interval: interval,
		sc:       sc,
	}
}

// Init triggers the first scan immediately.
func (m M) Init() tea.Cmd {
	return scanCmd(m.sc)
}

// Update handles incoming messages.
func (m M) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = applySize(m)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "r":
			if !m.scanning {
				m.scanning = true
				return m, scanCmd(m.sc)
			}

		case "up", "k":
			m.tbl.MoveUp(1)
		case "down", "j":
			m.tbl.MoveDown(1)
		}

	case ScanCompleteMsg:
		m.scanning = false
		m.lastScan = time.Now()
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.err = nil
			m.devices = msg.Devices
			m.tbl.SetRows(devicesToRows(m.devices))
		}
		return m, tea.Tick(m.interval, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})

	case TickMsg:
		if !m.scanning {
			m.scanning = true
			return m, scanCmd(m.sc)
		}
	}

	return m, nil
}

// View renders the full TUI string.
func (m M) View() string {
	return renderView(m)
}

// scanCmd returns a Cmd that runs a scan in the background and sends ScanCompleteMsg.
func scanCmd(sc *scanner.Scanner) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		devices, err := sc.Scan(ctx)
		return ScanCompleteMsg{Devices: devices, Err: err}
	}
}
