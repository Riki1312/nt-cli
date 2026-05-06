package cli

import (
	"context"
	"encoding/json"

	"github.com/Riki1312/nt-cli/internal/auth"
	"github.com/Riki1312/nt-cli/internal/mcp"
	"github.com/Riki1312/nt-cli/internal/output"
)

type app struct {
	ensureToken func(context.Context) (*auth.Token, error)
	login       func(context.Context) error
	logout      func() error
	callTool    func(context.Context, string, string, map[string]any) (*mcp.ToolResult, error)
	callToolRaw func(context.Context, string, string, map[string]any) (*mcp.ToolResult, json.RawMessage, error)
	listTools   func(context.Context, string) ([]string, error)
	print       func(any) error
	hint        func(string)
}

func newApp() app {
	return app{
		ensureToken: auth.EnsureValidToken,
		login:       auth.Login,
		logout:      auth.DeleteToken,
		callTool:    mcp.CallTool,
		callToolRaw: mcp.CallToolRaw,
		listTools:   mcp.ListTools,
		print:       output.Print,
		hint:        output.Hint,
	}
}
