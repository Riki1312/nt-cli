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

### Pages - `nt page <id> <verb>`

```bash
nt page <id> read                              # fetch page content and properties
nt page <id> set '<json-properties>'           # update properties
nt page <id> write '<markdown>'                # replace page content
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
| `read` | `notion-fetch` | Returns JSON with properties and Notion-flavored Markdown content |
| `set` | `notion-update-page` | `command: "update_properties"`, flat params |
| `write` | `notion-update-page` | `command: "replace_content"`. Rejects writes that delete child pages unless `allow_deleting_content: true` |
| `append` | Read via `notion-fetch`, then `notion-update-page` | Reads current content, concatenates, then uses `command: "replace_content"` |
| `create` | `notion-create-pages` | `parent: {"page_id": "<id>"}` |
| `move` | `notion-move-pages` | |
| `duplicate` | `notion-duplicate-page` | Async; returns new page ID immediately but content populates later |
| `comments` | `notion-get-comments` | Returns XML-formatted discussion threads |
| `comment` | `notion-create-comment` | Uses `rich_text` format for content |

### Databases - `nt db <id> <verb>`

The `<id>` is the database ID for `read`, or the data source ID (collection ID) for `create` and `update`. Use `nt db <id> read` to discover data source IDs from the schema output.

```bash
nt db <id> read                                # fetch database schema and info
nt db <id> query '<sql>' [--params ...]        # query rows with SQL (use _ as table name)
nt db <id> create --props '<json>' [content]   # create a row
nt db <id> update [--title "..."] [--schema '<sql>']  # update database schema
```

#### MCP mapping

| CLI | MCP tool | Notes |
|-----|----------|-------|
| `read` | `notion-fetch` | Returns schema, SQLite DDL, templates, views |
| `query` | `notion-query-data-sources` | SQL mode; use `_` as table name placeholder |
| `create` | `notion-create-pages` | `parent: {"data_source_id": "<id>"}` |
| `update` | `notion-update-data-source` | Uses SQL DDL statements for schema changes |

## Workspace Commands

```bash
nt search '<query>' [--type page] [--limit 10]  # search across workspace
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
| `whoami` | `notion-get-users` (`user_id: "self"`) |
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

**Write operations** (`set`, `write`, `append`, `create`, `move`, etc.) - confirmation with the affected resource:
```json
{"id": "abc123", "ok": true}
```

**`create` operations** - include the URL of the new page:
```json
{"id": "abc123", "url": "https://notion.so/abc123", "ok": true}
```

## Error Handling

Errors go to **stderr** as JSON. Stdout remains clean for piping.

```json
{"error": "page not found", "code": "NOT_FOUND"}
```

Exit codes:
- `0` - success
- `1` - general error
- `2` - authentication error (expired token, not logged in)
- `3` - not found
- `4` - rate limited (includes `retry_after` in error JSON)
- `5` - permission denied

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

# copy content from one page to another
nt page <src> read | jq -r '.content' | nt page <dst> write -

# search and preview titles
nt search "meeting notes" | jq -r '.[].title'

# create a page with content from stdin
echo "# New Page\n\nContent here" | nt page <parent-id> create --title "My Page" -

# query a database (use _ as the table name)
nt db <data-source-id> query "SELECT Name, Status FROM _ WHERE Status = 'Done'"

# query with parameterized values
nt db <data-source-id> query "SELECT * FROM _ WHERE Status = ?" --params "In Progress"
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--raw` | Print the raw MCP JSON-RPC response (for debugging) |
| `--verbose` | Print request/response details to stderr |
