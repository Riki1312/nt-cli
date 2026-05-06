# Current State

## Active Work

All core commands implemented and tested end-to-end. Codebase reviewed and cleaned up.
Latest review note, 2026-05-06: project feedback requested; no code changes made. `go test ./...` passes when run with `GOCACHE=/tmp/nt-cli-go-build` because the default Go cache path is outside the sandbox.
Latest implementation note, 2026-05-06: docs drift fixed for SQL database queries, transform golden tests added, and MCP/tool error classification now maps not found, auth, rate limit, and permission failures to documented exit codes.

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
- `replace '<old>' '<new>'` now also uses read-then-replace client-side, because direct hosted MCP replacement is not reliable for scoped edits
- targeted `replace` rejects empty match strings, and `--first` is invalid with `--page`
- `replace --page` rejects writes that delete child pages (safety feature from Notion)

## Next Steps

- Add unit tests for auth/OAuth helper behavior
- `--cursor` pagination flag for search
- More structured tests around page update flows

## Key Files

- `cmd/nt/main.go` - CLI entrypoint
- `internal/cli/` - Cobra commands (root, login, search, page, db, create, users, tools)
- `internal/mcp/` - MCP client (client.go, transport.go)
- `internal/auth/` - OAuth flow and token storage
- `internal/output/` - JSON output and error handling
- `internal/transform/` - Response transformers (search, page, db, create, comments)
