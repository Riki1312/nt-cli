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

// callAndPrintRaw calls a tool and prints the raw MCP tool result.
func callAndPrintRaw(ctx context.Context, a app, token, tool string, args map[string]any) error {
	result, data, err := a.callToolRaw(ctx, token, tool, args)
	if err != nil {
		return err
	}
	if result.IsError {
		return output.ToolError(result.TextContent())
	}
	return a.print(json.RawMessage(data))
}

// callTool calls a tool and checks for MCP tool errors, returning a
// structured CLI error with the right exit code on failure.
func callTool(ctx context.Context, a app, token, tool string, args map[string]any) (*mcp.ToolResult, error) {
	result, err := a.callTool(ctx, token, tool, args)
	if err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, output.ToolError(result.TextContent())
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

func requireNoArgs(verb string, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("%s does not accept arguments", verb)
	}
	return nil
}

func requireExactArgs(verb string, args []string, n int, usage string) error {
	if len(args) != n {
		return fmt.Errorf("%s requires %s", verb, usage)
	}
	return nil
}

func requireMaxArgs(verb string, args []string, n int, usage string) error {
	if len(args) > n {
		return fmt.Errorf("%s accepts at most %s", verb, usage)
	}
	return nil
}
