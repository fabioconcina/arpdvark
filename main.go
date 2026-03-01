package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/fabioconcina/arpdvark/exitcode"
	"github.com/fabioconcina/arpdvark/mcpserver"
	"github.com/fabioconcina/arpdvark/output"
	"github.com/fabioconcina/arpdvark/scanner"
	"github.com/fabioconcina/arpdvark/tags"
	"github.com/fabioconcina/arpdvark/tui"
)

var version = "dev"

func main() {
	var (
		ifaceName   string
		interval    int
		allowLarge  bool
		showVersion bool
		jsonOutput  bool
		onceOutput  bool
		mcpMode     bool
	)

	flag.StringVar(&ifaceName, "i", "", "Network interface to scan (default: auto-detect)")
	flag.IntVar(&interval, "t", 10, "Refresh interval in seconds")
	flag.BoolVar(&allowLarge, "large", false, "Allow scanning subnets larger than /16")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.BoolVar(&jsonOutput, "json", false, "Run one scan and output JSON to stdout")
	flag.BoolVar(&onceOutput, "once", false, "Run one scan and print a table to stdout")
	flag.BoolVar(&mcpMode, "mcp", false, "Run as MCP server (stdio transport)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: arpdvark [options]\n\nOptions:\n")
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

	// Single-shot modes: scan once and exit.
	if jsonOutput || onceOutput {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		devices, err := sc.Scan(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(exitcode.GeneralError)
		}
		if len(devices) == 0 {
			if jsonOutput {
				fmt.Println("[]")
			}
			os.Exit(exitcode.NoDevices)
		}
		allTags := store.All()
		if jsonOutput {
			if err := output.WriteJSON(os.Stdout, devices, allTags); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(exitcode.GeneralError)
			}
		} else {
			output.WriteTable(os.Stdout, devices, allTags)
		}
		return
	}

	// Default: interactive TUI.
	m := tui.New(sc, store, time.Duration(interval)*time.Second, version)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitcode.GeneralError)
	}
}
