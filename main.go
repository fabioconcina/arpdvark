package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/fabioconcina/arpdvark/activity"
	"github.com/fabioconcina/arpdvark/exitcode"
	"github.com/fabioconcina/arpdvark/mcpserver"
	"github.com/fabioconcina/arpdvark/notify"
	"github.com/fabioconcina/arpdvark/output"
	"github.com/fabioconcina/arpdvark/scanner"
	"github.com/fabioconcina/arpdvark/state"
	"github.com/fabioconcina/arpdvark/tags"
	"github.com/fabioconcina/arpdvark/tui"
)

var version = "dev"

func main() {
	// Handle subcommands before flag.Parse().
	if len(os.Args) >= 2 && os.Args[1] == "forget" {
		runForget(os.Args[2:])
		return
	}

	var (
		ifaceName   string
		interval    int
		allowLarge  bool
		showVersion bool
		jsonOutput  bool
		onceOutput  bool
		mcpMode     bool
		allDevices  bool
		notifyURL   string
	)

	flag.StringVar(&ifaceName, "i", "", "Network interface to scan (default: auto-detect)")
	flag.IntVar(&interval, "t", 10, "Refresh interval in seconds")
	flag.BoolVar(&allowLarge, "large", false, "Allow scanning subnets larger than /16")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.BoolVar(&jsonOutput, "json", false, "Run one scan and output JSON to stdout")
	flag.BoolVar(&onceOutput, "once", false, "Run one scan and print a table to stdout")
	flag.BoolVar(&mcpMode, "mcp", false, "Run as MCP server (stdio transport)")
	flag.BoolVar(&allDevices, "all", false, "Include offline devices (--json and --once only)")
	flag.StringVar(&notifyURL, "notify-url", "", "URL to POST when new devices are detected (e.g. ntfy.sh topic)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: arpdvark [options]\n       arpdvark forget [--older-than N] [MAC ...]\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if showVersion {
		fmt.Println("arpdvark", version)
		return
	}

	// Mutual exclusion: at most one output mode.
	modes := 0
	if jsonOutput {
		modes++
	}
	if onceOutput {
		modes++
	}
	if mcpMode {
		modes++
	}
	if modes > 1 {
		fmt.Fprintln(os.Stderr, "Error: --json, --once, and --mcp are mutually exclusive")
		os.Exit(exitcode.GeneralError)
	}

	if interval < 1 {
		fmt.Fprintln(os.Stderr, "Error: refresh interval must be at least 1 second")
		os.Exit(exitcode.GeneralError)
	}

	// MCP server creates scanners per-call, so start it before scanner.New()
	// to avoid requiring sudo just to launch the server.
	if mcpMode {
		if err := mcpserver.Run(version); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(exitcode.GeneralError)
		}
		return
	}

	sc, err := scanner.New(ifaceName, allowLarge)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "cannot open ARP socket") {
			fmt.Fprintf(os.Stderr, "Error: insufficient permissions. Try running with sudo.\n(%v)\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(exitcode.GeneralError)
	}

	store, err := tags.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load tags file: %v\n", err)
		store = tags.Empty()
	}

	stateStore, err := state.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load state file: %v\n", err)
		stateStore = state.Empty()
	}

	actStore, err := activity.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load activity file: %v\n", err)
		actStore = activity.Empty()
	}

	// Single-shot modes: scan once and exit.
	if jsonOutput || onceOutput {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		devices, err := sc.Scan(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(exitcode.GeneralError)
		}

		// Record activity for discovered devices.
		onlineMACs := make([]string, len(devices))
		for i, d := range devices {
			onlineMACs[i] = d.MAC.String()
		}
		actStore.Record(onlineMACs, time.Now())
		actStore.Save()

		// Notify for new devices (check before Merge adds them to state).
		if notifyURL != "" {
			var newDevices []scanner.Device
			for _, d := range devices {
				if !stateStore.Known(d.MAC.String()) {
					newDevices = append(newDevices, d)
				}
			}
			if err := notify.Send(notifyURL, newDevices); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: notification failed: %v\n", err)
			}
		}

		allTags := store.All()

		if allDevices {
			allDevs, mergeErr := stateStore.Merge(devices)
			if mergeErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", mergeErr)
				os.Exit(exitcode.GeneralError)
			}
			if len(allDevs) == 0 {
				if jsonOutput {
					fmt.Println("[]")
				}
				os.Exit(exitcode.NoDevices)
			}
			if jsonOutput {
				if err := output.WriteJSONFromState(os.Stdout, allDevs, allTags); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(exitcode.GeneralError)
				}
			} else {
				output.WriteTableFromState(os.Stdout, allDevs, allTags)
			}
		} else {
			// Still merge to update state file, but only display online devices.
			stateStore.Merge(devices)
			if len(devices) == 0 {
				if jsonOutput {
					fmt.Println("[]")
				}
				os.Exit(exitcode.NoDevices)
			}
			if jsonOutput {
				if err := output.WriteJSON(os.Stdout, devices, allTags); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(exitcode.GeneralError)
				}
			} else {
				output.WriteTable(os.Stdout, devices, allTags)
			}
		}
		return
	}

	// Default: interactive TUI.
	m := tui.New(sc, store, stateStore, actStore, time.Duration(interval)*time.Second, version, notifyURL)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitcode.GeneralError)
	}
}

func runForget(args []string) {
	fs := flag.NewFlagSet("forget", flag.ExitOnError)
	olderThan := fs.Int("older-than", 0, "Remove devices not seen in N days")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: arpdvark forget [--older-than N] [MAC ...]\n\nOptions:\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	stateStore, err := state.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitcode.GeneralError)
	}

	actStore, err := activity.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load activity file: %v\n", err)
		actStore = activity.Empty()
	}

	tagStore, err := tags.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load tags file: %v\n", err)
		tagStore = tags.Empty()
	}

	if *olderThan > 0 {
		cutoff := time.Now().AddDate(0, 0, -*olderThan)
		removed, err := stateStore.ForgetOlderThan(cutoff)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(exitcode.GeneralError)
		}
		for _, mac := range removed {
			actStore.Forget(mac)
			tagStore.Forget(mac)
		}
		actStore.Save()
		fmt.Printf("Removed %d device(s)\n", len(removed))
		return
	}

	macs := fs.Args()
	if len(macs) == 0 {
		fs.Usage()
		os.Exit(exitcode.GeneralError)
	}
	for _, mac := range macs {
		if err := stateStore.Forget(mac); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing %s: %v\n", mac, err)
			os.Exit(exitcode.GeneralError)
		}
		actStore.Forget(mac)
		tagStore.Forget(mac)
		fmt.Printf("Removed %s\n", mac)
	}
	actStore.Save()
}
