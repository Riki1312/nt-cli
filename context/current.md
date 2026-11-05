# Current State

## Active Work

Preparing v0.3.0. Structured discussion/comment reads, selected-content comments, explicit discussion replies, and own-account reactions are implemented. [Comments scope](comments.md) and [the command contract](../cli-design.md#comments-and-discussions) describe the boundary and JSON output change.

Live acceptance on an authorized scratch page verified page and inline comments, replies in the original discussion, and idempotent reaction add/remove. Actual response envelopes informed synthetic fixtures. Rendered comment bodies are opaque content; only line-framed metadata is parsed. Counts/cursors retain partial histories; availability-only responses have unknown completeness.

Pinned quality CI and tag-release gates are in place. Race tests, vet, formatting, and native builds pass. Final normalized live checks, minimum-Go validation, snapshot packaging, publication, and consumer installation remain. The published baseline is v0.2.1; no automatic conversation routing is enabled by this CLI release.

## What Works

### Page commands (`nt page <id> <verb>`)

- `read` - fetch page with parsed properties and content
- `set '<json>'` - update page properties
- `replace '<old>' '<new>'` - find-and-replace content (all matches)
- `replace --first '<old>' '<new>'` - replace only the first match
- `replace --page '<md>'` - replace entire page content
- `append '<md>'` - append to page (read-then-replace strategy)
- `create --title "..."` - create child page
- `move --to <id>` - move page to new parent
- `duplicate` - duplicate page (async)
- `comments` - structured discussions/comments with counts, authors, anchors, and continuation metadata
- `comment '<markdown>'` - add a page or selected-content comment

### Discussion and comment commands

- `discussion <url> read` - inspect a discussion; cursor reads fetch older comments
- `discussion <url> reply '<markdown>'` - reply within that discussion
- Bare discussion IDs require `--page PAGE_ID`
- `comment <id> react|unreact EMOJI` - set the connected user's own reaction state

### Database commands (`nt db <id> <verb>`)

- `read` - fetch database schema, SQLite DDL, templates, views
- `query '<sql>'` - query rows with SQL; use `_` as table name, `--params` for parameterized queries
- `create --props '<json>'` - create a row (uses data source ID)
- `update --title/--schema` - update database schema

### Workspace commands

- `search '<query>'` - workspace search (supports `--limit`, `--type`)
- `create --title "..."` - standalone page at workspace root
- `login` / `logout` - OAuth
- `whoami` / `users` / `teams` - user and team info

### Infrastructure

- All commands support `--raw` for raw MCP tool output
- Token refresh works automatically
- `logout` removes saved token and OAuth client registration files
- Hidden `tools` command for debugging available MCP tools

## Key Limitations

- `nt page <id> read` on a database returns empty content but includes a hint to use `nt db` instead
- `append` uses read-then-replace (not `insert_content_after`) due to selection matching fragility
- `replace '<old>' '<new>'` now also uses read-then-replace client-side, because direct hosted MCP replacement is not reliable for scoped edits
- targeted `replace` rejects empty match strings, and `--first` is invalid with `--page`
- `replace --page` rejects writes that delete child pages (safety feature from Notion)

## Next Steps

- `--cursor` pagination flag for search
- More command-level tests around page update edge cases if they catch behavior not already covered
- Consider true end-to-end smoke tests against a disposable Notion workspace when credentials/test workspace setup exists

## Key Files

- `cmd/nt/main.go` - CLI entrypoint
- `internal/cli/` - Cobra commands (root, login, search, page, db, create, users, tools)
- `internal/mcp/` - MCP client (client.go, transport.go)
- `internal/auth/` - OAuth flow and token storage
- `internal/output/` - JSON output and error handling
- `internal/transform/` - Response transformers (search, page, db, create, comments)
