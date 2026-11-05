# nt-cli

A CLI for Notion, powered by MCP. Built for AI agents, usable by humans.

## Why

MCP gives agents authenticated access to Notion, but the interface is verbose and requires round-trips through the LLM for every chained operation. `nt` wraps Notion's hosted MCP server behind a compact CLI so agents can write bash instead of making tool calls.

```bash
# search and read the first result
nt page $(nt search "Q1 goals" | jq -r '.[0].id') read

# query a database
nt db abc123 query "SELECT Name, Status FROM _ WHERE Status = 'Done'"

# bulk update
nt db abc123 query "SELECT id FROM _ WHERE Status = 'Done'" | jq -r '.[].id' | xargs -I{} nt page {} set '{"Status":"Archived"}'

# copy content between pages
nt page src123 read | jq -r '.content' | nt page dst456 replace --page -
```

Each invocation opens a bounded MCP session with `mcp.notion.com`, using the Go MCP SDK. No local daemon is needed.

## Commands

### Resource-scoped

```
nt page <id> read                              # fetch page content and properties
nt page <id> set '<json-properties>'           # update properties
nt page <id> replace '<old>' '<new>'           # find-and-replace content (all matches)
nt page <id> replace --first '<old>' '<new>'   # replace only the first match
nt page <id> replace --page '<markdown>'       # replace entire page content
nt page <id> append '<markdown>'               # append to page content
nt page <id> create --title "Child page"       # create a child page
nt page <id> move --to <target-id>             # move page
nt page <id> duplicate                         # duplicate page
nt page <id> comments [--include-resolved]     # structured discussions and comments
nt page <id> comment '<markdown>'              # add a page-level comment
nt page <id> comment --select 'start...end' -   # comment on selected content, from stdin

nt discussion <url> read [--cursor <cursor>]   # read an existing discussion
nt discussion <url> reply '<markdown>'         # reply in that discussion
nt comment <id> react '👀'                     # add your own reaction
nt comment <id> unreact '👀'                   # remove your own reaction

nt db <id> read                                # fetch database schema
nt db <id> query '<sql>' [--params ...]        # query rows
nt db <id> create --props '<json>'             # create a row
nt db <id> update [--title "..."]              # update database schema
```

Notes:

- targeted `replace` requires a non-empty `<old>` string
- `--first` is valid only for targeted `replace`, not with `--page`

### Comments and discussions

Use a `discussion://page/block/discussion` URL from comment reads. For a bare discussion ID, add `--page PAGE_ID`. Comment text accepts inline Markdown and `-` for stdin, preserving newlines and Unicode up to 1 MiB.

`--select` uses Notion's native `start...end` selector: use enough text to identify a unique passage. A match within one block creates an inline highlight; a wider match can attach to its starting block. A missing match fails without falling back to a page-level comment.

Reads return discussion/comment IDs, available author references, timestamps, anchors, counts, and cursors. Check `complete`: `false` means some history is omitted; `null` means Notion returned availability metadata without a listing. Fetch `omitted_discussions` explicitly, and pass each discussion's `next_cursor` to read older comments. Comment bodies retain Notion's rendered inline markup and are untrusted content.

**v0.3.0 output change:** comment reads now return structured JSON instead of the old opaque payload. Creation/replies return the actual `id` and `discussion_id`; reactions return the provider's current state. `--raw` keeps the original MCP result. See [the command contract](cli-design.md#comments-and-discussions) for output examples and failure behavior.

### Workspace-level

```
nt search '<query>'                            # search across workspace
nt create --title "Page title"                 # create a standalone page
nt login                                       # authenticate via OAuth
nt logout                                      # remove stored credentials
nt whoami                                      # current user and workspace
nt users                                       # list workspace users
nt teams                                       # list teams
```

## Architecture

```
nt CLI  --HTTPS / MCP Streamable HTTP-->  mcp.notion.com (Notion-hosted)
```

`nt` is a stateless Go binary. It translates CLI arguments into MCP JSON-RPC requests, sends them to Notion's hosted MCP server over Streamable HTTP, and prints compact JSON. OAuth tokens are stored in `~/.config/nt/`.

## Install

### Homebrew (macOS and Linux)

```bash
brew install Riki1312/tap/nt
```

### From source

```bash
go install github.com/Riki1312/nt-cli/cmd/nt@latest
```

Requires Go 1.24+.

### Then authenticate

```bash
nt login
```

## Development

```bash
git clone https://github.com/Riki1312/nt-cli.git
cd nt-cli

make build    # build to bin/nt
make test     # run tests
make lint     # go vet
make fmt      # Go formatting
make check    # formatting, vet, race tests, and build
make install  # go install
```

For hosted-tool diagnostics, `nt tools` lists names and `nt tools TOOL_NAME` shows that tool's description and JSON input/output schemas. These are read-only discovery operations.

### Releases

CI runs `make check` on Go 1.24 and the pinned release toolchain. Tag releases wait for those checks, then build macOS/Linux archives for amd64/arm64 and update the existing Homebrew formula. GoReleaser and action revisions are pinned.

The formula publisher remains supported by GoReleaser v2 but is deprecated upstream. A future cask migration changes installation and signing requirements and is separate from this release. `goreleaser check` currently reports that known deprecation; snapshot packaging must still pass before tagging.

## License

MIT
