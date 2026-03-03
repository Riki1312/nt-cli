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

## 2026-03-03: Database query now supported via hosted MCP

The `notion-query-data-sources` tool is now available on Notion's hosted MCP server (14 tools total, including `notion-query-data-sources` and `notion-query-meeting-notes`). The `nt db <id> query` command wraps this tool with SQL mode. Users write SQL using `_` as a table name placeholder, which is replaced with the full `collection://<id>` URL. Supports parameterized queries via `--params`.

## 2026-03-03: Client-side search filtering

Search results are filtered client-side via `--type` and `--limit` flags. The MCP `notion-search` tool does not support server-side filtering by type. Client-side filtering is acceptable because search results are bounded and the overhead is negligible compared to the API round-trip.

## 2026-03-03: Database type hint on page read

When `nt page <id> read` fetches a resource whose `metadata.type` is `"database"`, it includes a `hint` field in the JSON output and emits a hint to stderr suggesting `nt db <id> read`. This avoids a wasted round-trip when an agent or user accidentally uses the wrong command.
