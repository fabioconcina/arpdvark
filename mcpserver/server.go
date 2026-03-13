package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/fabioconcina/arpdvark/activity"
	"github.com/fabioconcina/arpdvark/output"
	"github.com/fabioconcina/arpdvark/scanner"
	"github.com/fabioconcina/arpdvark/tags"
)

// Run starts the MCP server on stdio and blocks until it exits.
func Run(version string) error {
	s := server.NewMCPServer(
		"arpdvark",
		version,
		server.WithToolCapabilities(true),
	)

	tool := mcp.NewTool("scan_network",
		mcp.WithDescription(
			"Scan the local network using ARP and return all discovered devices "+
				"with IP, MAC, vendor, hostname, and user-assigned labels.",
		),
		mcp.WithString("interface",
			mcp.Description("Network interface to scan (e.g. eth0). Auto-detected if omitted."),
		),
		mcp.WithBoolean("allow_large",
			mcp.Description("Allow scanning subnets larger than /16. Default false."),
		),
	)

	s.AddTool(tool, handleScan)

	return server.ServeStdio(s)
}

func handleScan(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ifaceName := req.GetString("interface", "")
	allowLarge := req.GetBool("allow_large", false)

	sc, err := scanner.New(ifaceName, allowLarge)
	if err != nil {
		return toolError(fmt.Sprintf("scanner init failed: %v", err)), nil
	}

	scanCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	devices, err := sc.Scan(scanCtx)
	if err != nil {
		return toolError(fmt.Sprintf("scan failed: %v", err)), nil
	}

	// Record activity history.
	onlineMACs := make([]string, len(devices))
	for i, d := range devices {
		onlineMACs[i] = d.MAC.String()
	}
	if actStore, err := activity.Load(); err == nil {
		actStore.Record(onlineMACs, time.Now())
		actStore.Save()
	}

	store, err := tags.Load()
	if err != nil {
		store = tags.Empty()
	}

	result := output.ToDeviceJSON(devices, store.All())
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("JSON encoding failed: %v", err)), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: msg,
			},
		},
		IsError: true,
	}
}
