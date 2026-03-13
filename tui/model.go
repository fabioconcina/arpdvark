package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/fabioconcina/arpdvark/activity"
	"github.com/fabioconcina/arpdvark/notify"
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

// SortColumn identifies which column to sort by.
type SortColumn int

const (
	SortIP SortColumn = iota
	SortMAC
	SortHostname
	SortLabel
	SortVendor
	SortLastSeen
	sortColumnCount // sentinel for wrapping
)

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

	sortCol SortColumn // current sort column
	sortAsc bool       // sort direction (true = ascending)

	filtering   bool           // whether filter input is active
	filterInput textinput.Model
	filterText  string // committed filter string

	knownMACs map[string]bool // MACs that existed in state before this session
	newMACs   map[string]bool // MACs discovered for the first time this session

	activityStore *activity.Store
	notifyURL     string
}

// New creates a new TUI model.
func New(sc *scanner.Scanner, store *tags.Store, stateStore *state.Store, actStore *activity.Store, interval time.Duration, version string, notifyURL string) M {
	ti := textinput.New()
	ti.Placeholder = "enter label (empty to clear)"
	ti.CharLimit = 64
	ti.Width = 40

	fi := textinput.New()
	fi.Placeholder = "filter..."
	fi.CharLimit = 64
	fi.Width = 40

	// Build set of already-known MACs so we can detect new ones.
	knownMACs := make(map[string]bool)
	for _, d := range stateStore.All() {
		knownMACs[d.MAC] = true
	}

	return M{
		tbl:         newTable(),
		iface:       sc.Interface(),
		subnet:      sc.Subnet(),
		interval:    interval,
		sc:          sc,
		version:     version,
		splash:      true,
		tagStore:    store,
		tags:        store.All(),
		editInput:   ti,
		filterInput: fi,
		stateStore:  stateStore,
		allDevices:  stateStore.All(),
		sortAsc:       true,
		knownMACs:     knownMACs,
		newMACs:       make(map[string]bool),
		activityStore: actStore,
		notifyURL:     notifyURL,
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

		if m.filtering {
			switch msg.String() {
			case "enter":
				m.filterText = strings.TrimSpace(m.filterInput.Value())
				m.filtering = false
				m.filterInput.Blur()
				refreshTable(&m)
				return m, nil
			case "esc":
				m.filtering = false
				m.filterInput.Blur()
				m.filterText = ""
				m.filterInput.SetValue("")
				refreshTable(&m)
				return m, nil
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
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
				m.activityStore.Save()
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.activityStore.Save()
			return m, tea.Quit

		case "esc":
			if m.filterText != "" {
				m.filterText = ""
				m.filterInput.SetValue("")
				refreshTable(&m)
				return m, nil
			}

		case "enter":
			devices := m.filteredDevices()
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
			refreshTable(&m)
			return m, nil

		case "s":
			m.sortAsc = !m.sortAsc
			refreshTable(&m)
			return m, nil

		case "left", "h":
			if m.sortCol > 0 {
				m.sortCol--
			} else {
				m.sortCol = sortColumnCount - 1
			}
			refreshTable(&m)
			return m, nil

		case "right", "l":
			m.sortCol++
			if m.sortCol >= sortColumnCount {
				m.sortCol = SortIP
			}
			refreshTable(&m)
			return m, nil

		case "/":
			if m.filterText != "" && !m.filtering {
				// clear filter if already active
				m.filterText = ""
				m.filterInput.SetValue("")
				refreshTable(&m)
				return m, nil
			}
			m.filterInput.SetValue(m.filterText)
			m.filterInput.Focus()
			m.filtering = true
			return m, textinput.Blink

		case "e":
			devices := m.filteredDevices()
			if len(devices) == 0 {
				return m, nil
			}
			mac := devices[m.tbl.Cursor()].MAC
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
			refreshTable(&m)
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
			// Detect new devices and notify.
			var newDevices []scanner.Device
			for _, d := range msg.Devices {
				mac := d.MAC.String()
				if !m.knownMACs[mac] {
					m.newMACs[mac] = true
					newDevices = append(newDevices, d)
				}
			}
			if m.notifyURL != "" && len(newDevices) > 0 {
				url := m.notifyURL
				go func() {
					if err := notify.Send(url, newDevices); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: notification failed: %v\n", err)
					}
				}()
			}
			// Record activity for online devices.
			onlineMACs := make([]string, len(msg.Devices))
			for i, d := range msg.Devices {
				onlineMACs[i] = d.MAC.String()
			}
			m.activityStore.Record(onlineMACs, time.Now())

			allDevs, err := m.stateStore.Merge(msg.Devices)
			if err != nil {
				m.err = err
			} else {
				m.allDevices = allDevs
			}
			refreshTable(&m)
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

// displayDevices returns the device list based on the showOffline toggle, sorted.
func (m M) displayDevices() []state.Device {
	var devices []state.Device
	if m.showOffline {
		devices = make([]state.Device, len(m.allDevices))
		copy(devices, m.allDevices)
	} else {
		devices = make([]state.Device, 0, len(m.allDevices))
		for _, d := range m.allDevices {
			if d.Online {
				devices = append(devices, d)
			}
		}
	}
	sortDevices(devices, m.sortCol, m.sortAsc, m.tags)
	return devices
}

// filteredDevices returns displayDevices filtered by the current filter text.
func (m M) filteredDevices() []state.Device {
	devices := m.displayDevices()
	if m.filterText == "" {
		return devices
	}
	filter := strings.ToLower(m.filterText)
	filtered := make([]state.Device, 0, len(devices))
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.IP), filter) ||
			strings.Contains(strings.ToLower(d.MAC), filter) ||
			strings.Contains(strings.ToLower(d.Hostname), filter) ||
			strings.Contains(strings.ToLower(m.tags[d.MAC]), filter) ||
			strings.Contains(strings.ToLower(d.Vendor), filter) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// refreshTable updates the table columns (sort indicator) and rows from current state.
func refreshTable(m *M) {
	updateColumns(m)
	m.tbl.SetRows(devicesToRows(m.filteredDevices(), m.tags))
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
