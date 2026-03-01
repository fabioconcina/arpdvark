package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"arpdvark/scanner"
	"arpdvark/tui"
)

func main() {
	var (
		ifaceName  string
		interval   int
		allowLarge bool
	)

	flag.StringVar(&ifaceName, "i", "", "Network interface to scan (default: auto-detect)")
	flag.IntVar(&interval, "t", 10, "Refresh interval in seconds")
	flag.BoolVar(&allowLarge, "large", false, "Allow scanning subnets larger than /16")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: arpdvark [options]\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

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

	m := tui.New(sc, time.Duration(interval)*time.Second)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
