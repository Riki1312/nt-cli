# Current State

## Active Work

All core commands implemented and tested end-to-end. Codebase reviewed and cleaned up.

## What Works

### Page commands (`nt page <id> <verb>`)
- `read` - fetch page with parsed properties and content
- `set '<json>'` - update page properties
- `write '<md>'` - replace page content
- `append '<md>'` - append to page (read-then-replace strategy)
- `create --title "..."` - create child page
- `move --to <id>` - move page to new parent
- `duplicate` - duplicate page (async)
- `comments` - list comments (XML format)
- `comment '<text>'` - add a comment

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
- All commands support `--raw` for raw MCP JSON output
- Token refresh works automatically
- Hidden `tools` command for debugging available MCP tools

## Key Limitations

- `nt page <id> read` on a database returns empty content but includes a hint to use `nt db` instead
- `append` uses read-then-replace (not `insert_content_after`) due to selection matching fragility
- `replace_content` rejects writes that delete child pages (safety feature from Notion)

## Next Steps

- Add tests (golden file tests for transforms, unit tests for auth)
- Error code mapping (not found, rate limited, permission denied -> exit codes)
- `--cursor` pagination flag for search
- `set` stdin support for properties JSON
- `write --replace` for targeted section replacement

## Key Files

- `cmd/nt/main.go` - CLI entrypoint
- `internal/cli/` - Cobra commands (root, login, search, page, db, create, users, tools)
- `internal/mcp/` - MCP client (client.go, transport.go)
- `internal/auth/` - OAuth flow and token storage
- `internal/output/` - JSON output and error handling
- `internal/transform/` - Response transformers (search, page, db, create, comments)
