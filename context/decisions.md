# Decisions

Append-only log of architectural and design decisions.

## 2026-03-02: Use Notion's hosted MCP server over Streamable HTTP

Instead of running a local MCP server as a daemon subprocess, the CLI talks directly to Notion's hosted MCP server at `mcp.notion.com/mcp` over Streamable HTTP. This eliminates daemon lifecycle management, unix sockets, and process orchestration. Each CLI invocation is a stateless HTTPS request.

See: `idea.md` (Architecture section)

## 2026-03-02: Resource-first command structure

Commands that operate on a specific resource use `nt <resource-type> <id> <verb>` (e.g., `nt page abc123 read`). This puts the target ID front and center, making it trivial to restrict agent access via hooks or permission rules.

See: `cli-design.md` (Command Structure section)

## 2026-03-02: JSON-only output, no custom query syntax

All output is JSON. Filters and sorts use Notion's native JSON format instead of a custom DSL. Agents can generate JSON natively; no parser to build or maintain.

See: `cli-design.md` (Output Format section)
