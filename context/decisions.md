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

## 2026-03-02: Append uses read-then-replace strategy

The `nt page <id> append` command reads the page first, concatenates existing content with new content, then calls `replace_content`. The alternative (`insert_content_after` with `selection_with_ellipsis`) requires exact text matching against page content, which is fragile and fails when using "..." alone as the selection pattern.

## 2026-03-02: Database query not supported

The `notion-query-data-sources` tool is not available on Notion's hosted MCP server at `mcp.notion.com/mcp`. Only 12 tools are exposed: search, fetch, create-pages, update-page, move-pages, duplicate-page, create-database, update-data-source, create-comment, get-comments, get-teams, get-users. Database row querying is only available through Claude's internal MCP integration, not the public hosted endpoint.
