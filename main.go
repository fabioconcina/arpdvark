package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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
	)

	flag.StringVar(&ifaceName, "i", "", "Network interface to scan (default: auto-detect)")
	flag.IntVar(&interval, "t", 10, "Refresh interval in seconds")
	flag.BoolVar(&allowLarge, "large", false, "Allow scanning subnets larger than /16")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: arpdvark [options]\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if showVersion {
		fmt.Println("arpdvark", version)
		return
	}

	if interval < 1 {
		fmt.Fprintln(os.Stderr, "Error: refresh interval must be at least 1 second")
		os.Exit(1)
	}

	sc, err := scanner.New(ifaceName, allowLarge)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "cannot open ARP socket") {
			fmt.Fprintf(os.Stderr, "Error: insufficient permissions. Try running with sudo.\n(%v)\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	store, err := tags.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load tags file: %v\n", err)
		store = tags.Empty()
	}

	m := tui.New(sc, store, time.Duration(interval)*time.Second, version)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
