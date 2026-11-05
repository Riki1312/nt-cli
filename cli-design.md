# CLI Design

The `nt` command structure, output format, and conventions.

## Command Structure

Commands are split into two categories: **resource-scoped** (operate on a specific page, database, discussion, or comment) and **workspace-level** (operate across the workspace).

```
nt <resource-type> <id> <verb> [flags]    # resource-scoped
nt <verb> [args] [flags]                  # workspace-level
```

Resource-scoped commands put the target ID front and center. This makes the target visible to callers enforcing command permissions. Notion permissions still apply to the connected account.

## Resource Commands

### Pages - `nt page <id> <verb>`

```bash
nt page <id> read                              # fetch page content and properties
nt page <id> set '<json-properties>'           # update properties
nt page <id> replace '<old>' '<new>'           # find-and-replace content (all matches)
nt page <id> replace --first '<old>' '<new>'   # replace only the first match
nt page <id> replace --page '<markdown>'       # replace entire page content
nt page <id> append '<markdown>'               # append to page content
nt page <id> create --title "Child page"       # create a child page under this page
nt page <id> move --to <target-id>             # move page to a new parent
nt page <id> duplicate                         # duplicate the page
nt page <id> comments                          # list comments
nt page <id> comment '<text>'                  # add a comment
```

#### MCP mapping

| CLI         | MCP tool                                           | Notes                                                                                                                                                                                                                                                                                                                                                                                                                   |
| ----------- | -------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `read`      | `notion-fetch`                                     | Returns JSON with properties and Notion-flavored Markdown content                                                                                                                                                                                                                                                                                                                                                       |
| `set`       | `notion-update-page`                               | `command: "update_properties"`, flat params                                                                                                                                                                                                                                                                                                                                                                             |
| `replace`   | `notion-fetch` + `notion-update-page`              | With two args: read current content, replace client-side, then `command: "replace_content"` with merged markdown. Default replaces all matches; `--first` replaces only the first match. Targeted replace requires a non-empty `old` string. With `--page`: full content replacement (`new_str` only). `--first` is invalid with `--page`. Rejects writes that delete child pages unless `allow_deleting_content: true` |
| `append`    | Read via `notion-fetch`, then `notion-update-page` | Reads current content, concatenates, then uses `command: "replace_content"`                                                                                                                                                                                                                                                                                                                                             |
| `create`    | `notion-create-pages`                              | `parent: {"page_id": "<id>"}`                                                                                                                                                                                                                                                                                                                                                                                           |
| `move`      | `notion-move-pages`                                |                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `duplicate` | `notion-duplicate-page`                            | Async; returns new page ID immediately but content populates later                                                                                                                                                                                                                                                                                                                                                      |
| `comments`  | `notion-get-comments`                              | Structured JSON with discussion/comment records, counts, and continuation metadata                                                                                                                                                                                                                                                                                                                                      |
| `comment`   | `notion-create-comment`                            | Uses `markdown`; `--select` adds native `selection_with_ellipsis` targeting                                                                                                                                                                                                                                                                                                                                             |

### Comments and discussions

```sh
nt page PAGE_ID comments [--include-resolved]
nt page PAGE_ID comment 'Comment'
nt page PAGE_ID comment --select 'Passage start...passage end' - < comment.md
nt discussion DISCUSSION_URL read [--cursor CURSOR] [--include-resolved]
nt discussion DISCUSSION_URL reply - < reply.md
nt discussion DISCUSSION_ID read --page PAGE_ID
nt discussion DISCUSSION_ID reply --page PAGE_ID 'Reply'
nt comment COMMENT_ID react '👀'
nt comment COMMENT_ID unreact '👀'
```

A `discussion://page/block/discussion` URL provides the page and discussion target. A bare discussion ID requires `--page`; a URL and `--page` cannot be combined. Reads use `notion-get-comments`; replies use `notion-create-comment` with that discussion ID. The CLI rejects irrelevant flags before authentication.

Comment/reply content supports the hosted tool's inline Markdown and mention syntax. Text arguments accept `-` for stdin, preserving valid nonempty UTF-8 up to 1 MiB. `--select` passes Notion's native `selection_with_ellipsis`: use the beginning and end of a unique passage, separated by `...`. A match within one block creates an inline highlight; otherwise the comment may attach to the block where the match starts. A missing match fails; nt does not fall back to a page comment.

Reactions use `notion-set-comment-reaction` to set the connected account's own state. Supply a standard Unicode emoji, not a shortcode. Repeating `react` or `unreact` is idempotent. Notion plan and permission restrictions remain errors. Reactions do not resolve discussions.

#### Read output

This structured output replaces the opaque comment payload in v0.3.0:

```json
{
  "suggested_edits_status": "included",
  "shown_count": 1,
  "total_count": 1,
  "omitted_truncated": false,
  "omitted_discussions": [],
  "discussions": [
    {
      "id": "discussion://page/block/discussion",
      "type": "comment",
      "context": "inline",
      "text_context": "Selected passage",
      "resolved": false,
      "comment_count": 1,
      "total_comment_count": 1,
      "comments": [
        {
          "id": "comment-id",
          "user_url": "user://author-id/author@example.invalid",
          "user_id": "author-id",
          "datetime": "2026-01-01T09:00:00.000Z",
          "by_you": true,
          "content": "<b>Please review</b><br />Second line."
        }
      ]
    }
  ],
  "complete": true
}
```

- Provider IDs, URLs, author references, timestamps, anchors, and resolution state are retained when supplied. `by_you` is the provider's own-account marker; `user_id` is extracted only from an explicit `user://` reference. These fields do not authorize a caller's workflow.
- Comment `content` is opaque rendered inline markup. It can contain unescaped XML-like text; do not treat it as trusted HTML or parse it for author metadata. Extra metadata attributes appear under `attributes`.
- `complete: false` means this response omits discussions or older comments. Fetch `omitted_discussions` by URL, and use a discussion's `next_cursor` for older comments. `omitted_truncated: true` means even the omitted-reference list is truncated. nt does not automatically fetch history.
- Per-discussion counts describe supplied comments and the whole thread. A final cursor page can still have fewer comments than the whole discussion; no next cursor does not make that response complete.
- Notion sometimes returns only availability metadata. In that case arrays are empty and `shown_count`, `total_count`, `omitted_truncated`, and `complete` are `null`. This is not evidence of an empty history. `complete: true` describes the supplied listing under the requested filters, not inaccessible or excluded resolved comments.
- Comment listings currently expose no reaction state. Reaction commands return that state separately.

#### Write output and uncertainty

Creation and replies return actual acknowledged identifiers:

```json
{ "id": "comment-id", "discussion_id": "discussion-id" }
```

Reaction commands return the provider's current list for that comment:

```json
{
  "comment_id": "comment-id",
  "reactions": [
    {
      "emoji": "👀",
      "reacted_by_current_user": true,
      "reactor_urls": ["user://author-id"]
    }
  ]
}
```

An empty `reactions` array is a confirmed empty list; missing data is an error. A reply acknowledgement naming another discussion is emitted for inspection and followed by `WRITE_UNCONFIRMED` on stderr.

MCP calls have a one-minute operation limit and are never replayed by nt. Token refresh has a separate 30-second limit. A lost response or unreadable acknowledgement may follow a completed write: inspect the discussion before posting again. Unsupported framing or inconsistent counts fail visibly. `--raw` bypasses normalization and returns the original MCP result, without another call or write.

### Databases - `nt db <id> <verb>`

The `<id>` is the database ID for `read`, or the data source ID (collection ID) for `create` and `update`. Use `nt db <id> read` to discover data source IDs from the schema output.

```bash
nt db <id> read                                # fetch database schema and info
nt db <id> query '<sql>' [--params ...]        # query rows with SQL (use _ as table name)
nt db <id> create --props '<json>' [content]   # create a row
nt db <id> update [--title "..."] [--schema '<sql>']  # update database schema
```

#### MCP mapping

| CLI      | MCP tool                    | Notes                                        |
| -------- | --------------------------- | -------------------------------------------- |
| `read`   | `notion-fetch`              | Returns schema, SQLite DDL, templates, views |
| `query`  | `notion-query-data-sources` | SQL mode; use `_` as table name placeholder  |
| `create` | `notion-create-pages`       | `parent: {"data_source_id": "<id>"}`         |
| `update` | `notion-update-data-source` | Uses SQL DDL statements for schema changes   |

## Workspace Commands

```bash
nt search '<query>' [--type page] [--limit 10]  # search across workspace
nt create --title "Page title"    # create a standalone page (workspace root)
nt login                          # OAuth flow (prints authorization URL)
nt logout                         # remove stored credentials
nt whoami                         # current user and workspace info
nt users                          # list workspace users
nt teams                          # list teams/teamspaces
```

#### MCP mapping

| CLI      | MCP tool                               |
| -------- | -------------------------------------- |
| `search` | `notion-search`                        |
| `create` | `notion-create-pages` (no parent)      |
| `whoami` | `notion-get-users` (`user_id: "self"`) |
| `users`  | `notion-get-users`                     |
| `teams`  | `notion-get-teams`                     |

## Output Format

### JSON by default

Command results are JSON on stdout. Page content is Notion-flavored Markdown; comment content retains rendered inline markup. Help and version use the standard CLI display.

**`nt search`** - array of results:

```json
[
  {
    "id": "abc123",
    "type": "page",
    "title": "Q1 Goals",
    "url": "https://notion.so/abc123"
  },
  {
    "id": "def456",
    "type": "database",
    "title": "Tasks",
    "url": "https://notion.so/def456"
  }
]
```

**`nt page <id> read`** - single object:

```json
{
  "id": "abc123",
  "title": "Q1 Goals",
  "url": "https://notion.so/abc123",
  "properties": {
    "Status": "In Progress",
    "Assignee": "Alice",
    "Date": "2026-01-15"
  },
  "content": "## Overview\n\nOur Q1 goals are..."
}
```

**Write operations** (`set`, `replace`, `append`, `create`, `move`, etc.) - confirmation with the affected resource:

```json
{ "id": "abc123", "ok": true }
```

**`create` operations** - return an array because the MCP tool can create multiple pages:

```json
[{ "id": "abc123", "url": "https://notion.so/abc123", "ok": true }]
```

## Error Handling

Errors go to **stderr** as JSON. Stdout contains only result JSON; an unconfirmed reply can include the actual acknowledged IDs for inspection.

```json
{ "error": "page not found", "code": "NOT_FOUND" }
```

Exit codes:

- `0` - success
- `1` - general error
- `2` - authentication error (expired token, not logged in)
- `3` - not found
- `4` - rate limited
- `5` - permission denied

## Stdin Support

Content arguments accept `-` to read from stdin. Useful for long content or piping between commands.

```bash
# write content from a file
cat report.md | nt page <id> replace --page -

# pipe content between commands
nt page <source-id> read | jq -r '.content' | nt page <target-id> replace --page -

# properties from a file
cat props.json | nt page <id> set -
```

## Composition Examples

```bash
# search, grab first result, read it
nt page $(nt search "Q1 goals" | jq -r '.[0].id') read

# copy content from one page to another
nt page <src> read | jq -r '.content' | nt page <dst> replace --page -

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

| Flag    | Description                                   |
| ------- | --------------------------------------------- |
| `--raw` | Print the raw MCP tool result (for debugging) |
