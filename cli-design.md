# CLI Design

The `nt` command structure, output format, and conventions.

## Command Structure

Commands are split into two categories: **resource-scoped** (operate on a specific page or database) and **workspace-level** (operate across the workspace).

```
nt <resource-type> <id> <verb> [flags]    # resource-scoped
nt <verb> [args] [flags]                  # workspace-level
```

Resource-scoped commands put the target ID front and center. This makes it trivial to restrict agent access: allow `nt page abc123 *` and deny everything else.

## Resource Commands

### Pages -`nt page <id> <verb>`

```bash
nt page <id> read                              # fetch page content and properties
nt page <id> set '<json-properties>'           # update properties
nt page <id> write '<markdown>'                # replace page content
nt page <id> write --replace '<old>' '<new>'   # replace a section of page content
nt page <id> append '<markdown>'               # append to page content
nt page <id> create --title "Child page"       # create a child page under this page
nt page <id> move --to <target-id>             # move page to a new parent
nt page <id> duplicate                         # duplicate the page
nt page <id> comments                          # list comments
nt page <id> comment '<text>'                  # add a comment
```

#### MCP mapping

| CLI | MCP tool | Notes |
|-----|----------|-------|
| `read` | `notion-fetch` | |
| `set` | `notion-update-page` | `command: "update_properties"` |
| `write` | `notion-update-page` | `command: "replace_content"` or `"replace_content_range"` with `--replace` |
| `append` | `notion-update-page` | `command: "append"` |
| `create` | `notion-create-pages` | `parent: {"page_id": "<id>"}` |
| `move` | `notion-move-pages` | |
| `duplicate` | `notion-duplicate-page` | |
| `comments` | `notion-get-comments` | |
| `comment` | `notion-create-comment` | |

### Databases -`nt db <id> <verb>`

```bash
nt db <id> read                                         # fetch database schema and info
nt db <id> query [--filter '<json>'] [--sort '<json>']  # query rows
nt db <id> create --props '<json>' [--content '<md>']   # create a row
nt db <id> update [--title "..."] [--schema '<json>']   # update database schema
```

#### MCP mapping

| CLI | MCP tool | Notes |
|-----|----------|-------|
| `read` | `notion-fetch` | Returns schema, templates, metadata |
| `query` | `notion-query-data-sources` | Falls back to `notion-query-database-view` |
| `create` | `notion-create-pages` | `parent: {"data_source_id": "<id>"}` |
| `update` | `notion-update-data-source` | |

## Workspace Commands

```bash
nt search '<query>'               # search across workspace
nt create --title "Page title"    # create a standalone page (workspace root)
nt login                          # OAuth flow (opens browser)
nt logout                         # remove stored credentials
nt whoami                         # current user and workspace info
nt users                          # list workspace users
nt teams                          # list teams/teamspaces
```

#### MCP mapping

| CLI | MCP tool |
|-----|----------|
| `search` | `notion-search` |
| `create` | `notion-create-pages` (no parent) |
| `whoami` | `notion-get-self` |
| `users` | `notion-get-users` |
| `teams` | `notion-get-teams` |

## Output Format

### JSON by default

All commands output JSON to stdout. No exceptions. Content fields contain Notion-flavored Markdown as strings.

**`nt search`** - array of results:
```json
[
  {"id": "abc123", "type": "page", "title": "Q1 Goals", "url": "https://notion.so/abc123"},
  {"id": "def456", "type": "database", "title": "Tasks", "url": "https://notion.so/def456"}
]
```

**`nt page <id> read`** - single object:
```json
{
  "id": "abc123",
  "title": "Q1 Goals",
  "url": "https://notion.so/abc123",
  "properties": {"Status": "In Progress", "Assignee": "Alice", "Date": "2026-01-15"},
  "content": "## Overview\n\nOur Q1 goals are..."
}
```

**`nt db <id> query`** - array of rows:
```json
[
  {"id": "row1", "properties": {"Title": "Ship v2", "Status": "Done", "Date": "2026-02-01"}},
  {"id": "row2", "properties": {"Title": "Write docs", "Status": "In Progress", "Date": "2026-02-15"}}
]
```

**Write operations** (`set`, `write`, `append`, `create`, `move`, etc.) - confirmation with the affected resource:
```json
{"id": "abc123", "ok": true}
```

### Pagination

- `--limit <n>` -cap the number of results (default: 100)
- `--cursor <token>` -resume from a pagination cursor
- Paginated responses include a `next_cursor` field when there are more results:

```json
{
  "results": [...],
  "next_cursor": "eyJz..."
}
```

When results fit in a single page, the response is a flat array (no wrapper object).

## Error Handling

Errors go to **stderr** as JSON. Stdout remains clean for piping.

```json
{"error": "page not found", "code": "NOT_FOUND"}
```

Exit codes:
- `0` -success
- `1` -general error
- `2` -authentication error (expired token, not logged in)
- `3` -not found
- `4` -rate limited (includes `retry_after` in error JSON)
- `5` -permission denied

## Stdin Support

Content arguments accept `-` to read from stdin. Useful for long content or piping between commands.

```bash
# write content from a file
cat report.md | nt page <id> write -

# pipe content between commands
nt page <source-id> read | jq -r '.content' | nt page <target-id> write -

# properties from a file
cat props.json | nt page <id> set -
```

## Composition Examples

```bash
# search, grab first result, read it
nt page $(nt search "Q1 goals" | jq -r '.[0].id') read

# query a database, get IDs of done tasks
nt db <id> query --filter '{"property":"Status","status":{"equals":"Done"}}' | jq -r '.[].id'

# bulk update: mark all tasks in a list as done
nt db <id> query | jq -r '.[].id' | xargs -I{} nt page {} set '{"Status": "Done"}'

# copy content from one page to another
nt page <src> read | jq -r '.content' | nt page <dst> write -

# search and preview titles
nt search "meeting notes" | jq -r '.[].title'

# create a page with content from stdin
echo "# New Page\n\nContent here" | nt page <parent-id> create --title "My Page" -
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Force JSON output (useful if human-readable mode is added later) |
| `--raw` | Print the raw MCP JSON-RPC response (for debugging) |
| `--limit <n>` | Max results for list/query commands |
| `--cursor <token>` | Pagination cursor |
| `--verbose` | Print request/response details to stderr |
