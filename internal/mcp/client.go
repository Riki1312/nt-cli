package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const clientName = "nt-cli"

var clientVersion = "dev"

// ToolResult wraps the MCP tool call result for easier consumption.
type ToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError"`
}

// ContentItem is one MCP content block after conversion to nt's compact shape.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// TextContent extracts the first text content from a ToolResult.
// Returns empty string if no text content is found.
func (r *ToolResult) TextContent() string {
	for _, c := range r.Content {
		if c.Type == "text" {
			return c.Text
		}
	}
	return ""
}

// withSession connects to the Notion MCP server and calls fn with the session.
// The session is closed after fn returns.
func withSession(ctx context.Context, accessToken string, fn func(*sdkmcp.ClientSession) error) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: clientName, Version: clientVersion},
		nil,
	)

	transport := &sdkmcp.StreamableClientTransport{
		Endpoint:   NotionMCPEndpoint,
		HTTPClient: NewAuthenticatedHTTPClient(accessToken),
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("connecting to Notion MCP: %w", err)
	}
	defer session.Close()

	return fn(session)
}

// CallTool connects to the Notion MCP server, calls the named tool with the
// given arguments, and returns the result.
func CallTool(ctx context.Context, accessToken, toolName string, args map[string]any) (*ToolResult, error) {
	var tr *ToolResult
	err := withSession(ctx, accessToken, func(session *sdkmcp.ClientSession) error {
		result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		})
		if err != nil {
			return fmt.Errorf("calling tool %s: %w", toolName, err)
		}
		tr, err = convertResult(result)
		return err
	})
	return tr, err
}

// CallToolRaw is like CallTool but also returns the MCP tool result as JSON bytes.
func CallToolRaw(ctx context.Context, accessToken, toolName string, args map[string]any) (*ToolResult, json.RawMessage, error) {
	var tr *ToolResult
	var raw json.RawMessage
	err := withSession(ctx, accessToken, func(session *sdkmcp.ClientSession) error {
		result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		})
		if err != nil {
			return fmt.Errorf("calling tool %s: %w", toolName, err)
		}

		tr, err = convertResult(result)
		if err != nil {
			return err
		}

		raw, err = json.Marshal(result)
		if err != nil {
			return fmt.Errorf("marshaling result: %w", err)
		}
		return nil
	})
	return tr, raw, err
}

// ToolDefinition describes a hosted tool and its JSON input/output schemas.
type ToolDefinition struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	InputSchema  any    `json:"inputSchema"`
	OutputSchema any    `json:"outputSchema,omitempty"`
}

// ListTools discovers up to 1,000 hosted tools across all result pages.
// Discovery has a one-minute deadline and fails instead of returning a partial list.
func ListTools(ctx context.Context, accessToken string) ([]ToolDefinition, error) {
	tools := []ToolDefinition{}
	err := withSession(ctx, accessToken, func(session *sdkmcp.ClientSession) error {
		for tool, err := range session.Tools(ctx, nil) {
			if err != nil {
				return fmt.Errorf("listing tools: %w", err)
			}
			if len(tools) == 1000 {
				return fmt.Errorf("tool discovery exceeds 1,000 results")
			}
			tools = append(tools, ToolDefinition{
				Name: tool.Name, Description: tool.Description,
				InputSchema: tool.InputSchema, OutputSchema: tool.OutputSchema,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tools, nil
}

func convertResult(result *sdkmcp.CallToolResult) (*ToolResult, error) {
	tr := &ToolResult{
		IsError: result.IsError,
	}
	for _, c := range result.Content {
		switch v := c.(type) {
		case *sdkmcp.TextContent:
			tr.Content = append(tr.Content, ContentItem{Type: "text", Text: v.Text})
		default:
			data, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("marshaling content: %w", err)
			}
			tr.Content = append(tr.Content, ContentItem{Type: "unknown", Text: string(data)})
		}
	}
	return tr, nil
}
