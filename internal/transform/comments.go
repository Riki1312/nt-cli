package transform

import (
	"github.com/Riki1312/nt-cli/internal/mcp"
)

// Comments extracts comment text from a notion-get-comments result.
// The response is XML-formatted text; we return it as-is in a wrapper
// since parsing the XML is fragile and the raw text is already readable.
func Comments(result *mcp.ToolResult) any {
	text := result.TextContent()
	if text == "" {
		return map[string]any{"comments": ""}
	}
	return map[string]any{"comments": text}
}
