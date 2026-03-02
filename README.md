# nt-cli

A CLI for Notion, powered by MCP. Built for AI agents, usable by humans.

## Why

MCP gives agents authenticated access to Notion, but the interface is verbose and requires round-trips through the LLM for every chained operation. `nt` wraps Notion's hosted MCP server behind a compact CLI so agents can write bash instead of making tool calls.

```bash
# search and read the first result
nt page $(nt search "Q1 goals" | jq -r '.[0].id') read

# query a database
nt db abc123 query --filter '{"property":"Status","status":{"equals":"Done"}}'

# bulk update
nt db abc123 query | jq -r '.[].id' | xargs -I{} nt page {} set '{"Status":"Archived"}'

# copy content between pages
nt page src123 read | jq -r '.content' | nt page dst456 write -
```

No daemon. No SDK. Each invocation is a single HTTPS request to `mcp.notion.com`.

## Commands

### Resource-scoped

```
nt page <id> read                              # fetch page content and properties
nt page <id> set '<json-properties>'           # update properties
nt page <id> write '<markdown>'                # replace page content
nt page <id> write --replace '<old>' '<new>'   # replace a section of page content
nt page <id> append '<markdown>'               # append to page content
nt page <id> create --title "Child page"       # create a child page
nt page <id> move --to <target-id>             # move page
nt page <id> duplicate                         # duplicate page
nt page <id> comments                          # list comments
nt page <id> comment '<text>'                  # add a comment

nt db <id> read                                # fetch database schema
nt db <id> query [--filter '<json>']           # query rows
nt db <id> create --props '<json>'             # create a row
nt db <id> update [--title "..."]              # update database schema
```

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

```bash
go install github.com/Riki1312/nt-cli/cmd/nt@latest
```

Then authenticate:

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
make install  # go install
```

Requires Go 1.24+.

## License

MIT
