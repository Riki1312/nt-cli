package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const clientName = "nt-cli"
const clientVersion = "0.1.0"

// ToolResult wraps the MCP tool call result for easier consumption.
type ToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError"`
}

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

// CallToolRaw is like CallTool but also returns the raw MCP result as JSON bytes.
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
	var names []string
	err := withSession(ctx, accessToken, func(session *sdkmcp.ClientSession) error {
		result, err := session.ListTools(ctx, &sdkmcp.ListToolsParams{})
		if err != nil {
			return fmt.Errorf("listing tools: %w", err)
		}
		for _, t := range result.Tools {
			names = append(names, t.Name)
		}
		return nil
	})
	return names, err
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
