package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Riki1312/nt-cli/internal/mcp"
	"github.com/Riki1312/nt-cli/internal/output"
)

// callAndPrintRaw calls a tool and prints the raw JSON result, returning an
// error (with proper exit code) if the MCP tool reports a failure.
func callAndPrintRaw(ctx context.Context, token, tool string, args map[string]any) error {
	result, data, err := mcp.CallToolRaw(ctx, token, tool, args)
	if err != nil {
		return err
	}
	if result.IsError {
		return output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
	}
	return output.Print(json.RawMessage(data))
}

// callTool calls a tool and checks for MCP tool errors, returning a
// structured CLI error with the right exit code on failure.
func callTool(ctx context.Context, token, tool string, args map[string]any) (*mcp.ToolResult, error) {
	result, err := mcp.CallTool(ctx, token, tool, args)
	if err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, output.NewError(output.ExitError, "TOOL_ERROR", result.TextContent())
	}
	return result, nil
}

// readContentArg reads content from the argument or from stdin if the argument is "-".
func readContentArg(arg string) (string, error) {
	if arg == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading from stdin: %w", err)
		}
		return string(data), nil
	}
	return arg, nil
}
