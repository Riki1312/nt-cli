# Current State

## Current Work

The v0.3.0 implementation provides structured discussion/comment reads, selected-content comments, discussion replies, and own-account reactions. [Comments scope](comments.md) and [the command contract](../cli-design.md#comments-and-discussions) own the boundary and JSON output change. [Acceptance evidence](sessions/2026-11-05.md) records verification and limitations.

An authorized scratch page verified page/inline comments, replies in the original discussion, multiline Unicode content, idempotent reaction add/remove, normalized receipts, and a failed selector without a page-level fallback. Synthetic fixtures cover partial histories and malformed responses. Metadata-only responses retain unknown completeness; rendered bodies remain opaque content.

Formatting, vet, race tests on Go 1.24.0 and the release toolchain, and native builds pass. Snapshot packaging produced all four platform archives and the existing Homebrew formula. Pinned CI gates tag publication. The formula publisher has a known GoReleaser v2 deprecation; cask migration is separate.

Release publication and consumer installation are operational steps, not implied by source documentation. This CLI does not enable automatic Notion conversation routing.

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
