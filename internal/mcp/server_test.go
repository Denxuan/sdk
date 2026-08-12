package mcp

import (
	"context"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestProtocolServerExposesReadOnlyTools(t *testing.T) {
	ctx := context.Background()
	server := NewServer(t.TempDir()).protocolServer()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) < 8 {
		t.Fatalf("expected at least 8 tools, got %d", len(result.Tools))
	}
	for _, tool := range result.Tools {
		if tool.Description == "" {
			t.Fatalf("tool %q has no description", tool.Name)
		}
	}
	call, err := clientSession.CallTool(ctx, &sdkmcp.CallToolParams{Name: "sdk_current_versions"})
	if err != nil {
		t.Fatal(err)
	}
	if call.IsError || call.StructuredContent == nil {
		t.Fatalf("unexpected current response: %#v", call)
	}
}
