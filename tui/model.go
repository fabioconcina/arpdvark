package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/fabioconcina/arpdvark/scanner"
	"github.com/fabioconcina/arpdvark/state"
	"github.com/fabioconcina/arpdvark/tags"
)

// ScanCompleteMsg is sent when a scan finishes (successfully or not).
type ScanCompleteMsg struct {
	Devices []scanner.Device
	Err     error
}

// TickMsg is sent by the refresh timer to trigger the next scan.
type TickMsg time.Time

// splashDoneMsg is sent after the splash screen timer expires.
type splashDoneMsg struct{}

// labelSavedMsg is sent after a tag write completes (success or error).
type labelSavedMsg struct {
	mac   string
	label string
	err   error
}

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
	version  string
	splash   bool

	tagStore  *tags.Store
	tags      map[string]string // mac → label, mirrored from tagStore
	editing   bool
	editInput textinput.Model
	editMAC   string

	stateStore  *state.Store
	allDevices  []state.Device // all known devices (online + offline)
	showOffline bool           // toggle: show/hide offline devices

	detailView   bool
	detailDevice state.Device
	detailCursor int // selected field index in detail view
}

// New creates a new TUI model.
func New(sc *scanner.Scanner, store *tags.Store, stateStore *state.Store, interval time.Duration, version string) M {
	ti := textinput.New()
	ti.Placeholder = "enter label (empty to clear)"
	ti.CharLimit = 64
	ti.Width = 40

	return M{
		tbl:        newTable(),
		iface:      sc.Interface(),
		subnet:     sc.Subnet(),
		interval:   interval,
		sc:         sc,
		version:    version,
		splash:     true,
		tagStore:   store,
		tags:       store.All(),
		editInput:  ti,
		stateStore: stateStore,
		allDevices: stateStore.All(),
	}
}

// Init shows the splash screen for 2 seconds before starting the first scan.
func (m M) Init() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return splashDoneMsg{}
	})
}

// Update handles incoming messages.
func (m M) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case splashDoneMsg:
		m.splash = false
		m.scanning = true
		return m, scanCmd(m.sc)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = applySize(m)

	case tea.KeyMsg:
		if m.editing {
			switch msg.String() {
			case "enter":
				label := strings.TrimSpace(m.editInput.Value())
				mac := m.editMAC
				store := m.tagStore
				return m, func() tea.Msg {
					err := store.Set(mac, label)
					return labelSavedMsg{mac: mac, label: label, err: err}
				}
			case "esc":
				m.editing = false
				m.editInput.Blur()
				m.editMAC = ""
				return m, nil
			default:
				var cmd tea.Cmd
				m.editInput, cmd = m.editInput.Update(msg)
				return m, cmd
			}
		}

		if m.detailView {
			switch msg.String() {
			case "esc", "enter":
				m.detailView = false
				return m, nil
			case "up", "k":
				if m.detailCursor > 0 {
					m.detailCursor--
				}
				return m, nil
			case "down", "j":
				if m.detailCursor < detailFieldCount-1 {
					m.detailCursor++
				}
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "enter":
			devices := m.displayDevices()
			if len(devices) == 0 {
				return m, nil
			}
			m.detailDevice = devices[m.tbl.Cursor()]
			m.detailCursor = 0
			m.detailView = true
			return m, nil

		case "r":
			if !m.scanning {
				m.scanning = true
				return m, scanCmd(m.sc)
			}

		case "o":
			m.showOffline = !m.showOffline
			m.tbl.SetRows(devicesToRows(m.displayDevices(), m.tags))
			return m, nil

		case "e":
			if len(m.displayDevices()) == 0 {
				return m, nil
			}
			sel := m.tbl.SelectedRow()
			if sel == nil {
				return m, nil
			}
			mac := sel[1]
			m.editMAC = mac
			m.editInput.SetValue(m.tags[mac])
			m.editInput.Focus()
			m.editing = true
			return m, textinput.Blink

		case "up", "k":
			m.tbl.MoveUp(1)
		case "down", "j":
			m.tbl.MoveDown(1)
		}

	case labelSavedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			if msg.label == "" {
				delete(m.tags, msg.mac)
			} else {
				m.tags[msg.mac] = msg.label
			}
			m.tbl.SetRows(devicesToRows(m.displayDevices(), m.tags))
		}
		m.editing = false
		m.editInput.Blur()
		m.editMAC = ""

	case ScanCompleteMsg:
		m.scanning = false
		m.lastScan = time.Now()
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.err = nil
			m.devices = msg.Devices
			allDevs, err := m.stateStore.Merge(msg.Devices)
			if err != nil {
				m.err = err
			} else {
				m.allDevices = allDevs
			}
			m.tbl.SetRows(devicesToRows(m.displayDevices(), m.tags))
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
	if m.splash {
		return renderSplash(m)
	}
	if m.detailView {
		return renderDetail(m)
	}
	return renderView(m)
}

// displayDevices returns the device list to show based on the showOffline toggle.
func (m M) displayDevices() []state.Device {
	if m.showOffline {
		return m.allDevices
	}
	online := make([]state.Device, 0, len(m.allDevices))
	for _, d := range m.allDevices {
		if d.Online {
			online = append(online, d)
		}
	}
	return online
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
