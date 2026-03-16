package mcpserver

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestMCPServer_InitializeAndListTools(t *testing.T) {
	srv := NewServer("0.0.1-test")

	c, err := client.NewInProcessClient(srv)
	if err != nil {
		t.Fatalf("creating client: %v", err)
	}

	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("starting client: %v", err)
	}
	defer c.Close()

	// Initialize
	initResult, err := c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			ClientInfo:      mcp.Implementation{Name: "test-client", Version: "0.0.1"},
		},
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}

	if initResult.ServerInfo.Name != "arpdvark" {
		t.Errorf("server name = %q, want %q", initResult.ServerInfo.Name, "arpdvark")
	}
	if initResult.ServerInfo.Version != "0.0.1-test" {
		t.Errorf("server version = %q, want %q", initResult.ServerInfo.Version, "0.0.1-test")
	}

	// List tools
	toolsResult, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	if len(toolsResult.Tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(toolsResult.Tools))
	}

	tool := toolsResult.Tools[0]
	if tool.Name != "scan_network" {
		t.Errorf("tool name = %q, want %q", tool.Name, "scan_network")
	}
}
