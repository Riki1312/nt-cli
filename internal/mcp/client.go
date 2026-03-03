package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolResult wraps the MCP tool call result for easier consumption.
type ToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// CallTool connects to the Notion MCP server, calls the named tool with the
// given arguments, and returns the result. The session is closed before returning.
func CallTool(ctx context.Context, accessToken, toolName string, args map[string]any) (*ToolResult, error) {
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "nt-cli", Version: "0.1.0"},
		nil,
	)

	transport := &sdkmcp.StreamableClientTransport{
		Endpoint:   NotionMCPEndpoint,
		HTTPClient: NewAuthenticatedHTTPClient(accessToken),
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to Notion MCP: %w", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		return nil, fmt.Errorf("calling tool %s: %w", toolName, err)
	}

	return convertResult(result)
}

// CallToolRaw is like CallTool but returns the raw MCP result as JSON bytes.
// The ToolResult is also returned so callers can check IsError.
func CallToolRaw(ctx context.Context, accessToken, toolName string, args map[string]any) (*ToolResult, json.RawMessage, error) {
	result, err := CallTool(ctx, accessToken, toolName, args)
	if err != nil {
		return nil, nil, err
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling result: %w", err)
	}
	return result, raw, nil
}

// ListTools connects to the Notion MCP server and returns the available tool names.
func ListTools(ctx context.Context, accessToken string) ([]string, error) {
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "nt-cli", Version: "0.1.0"},
		nil,
	)

	transport := &sdkmcp.StreamableClientTransport{
		Endpoint:   NotionMCPEndpoint,
		HTTPClient: NewAuthenticatedHTTPClient(accessToken),
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to Notion MCP: %w", err)
	}
	defer session.Close()

	result, err := session.ListTools(ctx, &sdkmcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("listing tools: %w", err)
	}

	var names []string
	for _, t := range result.Tools {
		names = append(names, t.Name)
	}
	return names, nil
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
			// Marshal unknown content types as JSON text
			data, _ := json.Marshal(v)
			tr.Content = append(tr.Content, ContentItem{Type: "unknown", Text: string(data)})
		}
	}
	return tr, nil
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
