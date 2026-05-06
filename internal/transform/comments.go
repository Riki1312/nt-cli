package transform

import (
	"github.com/Riki1312/nt-cli/internal/mcp"
)

// CommentResult is the raw comment thread payload from Notion MCP.
type CommentResult struct {
	Comments string `json:"comments"`
}

// Comments extracts comment text from a notion-get-comments result.
// The response is XML-formatted text; we return it as-is in a wrapper
// since parsing the XML is fragile and the raw text is already readable.
func Comments(result *mcp.ToolResult) CommentResult {
	return CommentResult{Comments: result.TextContent()}
}
