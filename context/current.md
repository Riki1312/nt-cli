# Current State

## Active Work

Preparing v0.3.0 for structured comment reads, selected-content comments, discussion replies, and own-account reactions. [Comments design](comments.md) owns the requested behavior and release scope. The hosted MCP schemas were inspected using a temporary read-only probe; consumer installations remain unchanged.

Schema discovery now supports `nt tools NAME` and paginated tool listing with a bounded MCP call lifetime. This slice passes race tests, vet, formatting, and builds on Go 1.24 and the release toolchain. Request-side selected comments, discussion reads/replies, and own-account reactions are implemented and tested against the live input contracts. Reads still use the existing opaque comment payload; structured normalization and live acceptance remain unfinished. The hosted tools publish input schemas but no output schemas; a read-only check of an empty page returned suggestion-status metadata alone. An agreed scratch page is now available for observing a controlled nonempty discussion and verifying selected comments/replies/reactions before response normalization and publication.

Quality CI and tag-release gates are prepared with pinned action/tool revisions. Snapshot packaging produced all four archives and the existing Homebrew formula without publishing. The formula publisher has a documented GoReleaser v2 deprecation; archive syntax is updated.

The v0.2.1 baseline remains implemented below. Tests and vet passed before this work.

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
