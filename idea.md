# nt-cli: A CLI for Notion, powered by MCP

## The Problem

The Notion MCP server gives AI agents authenticated access to a Notion workspace (search, read, create, update, move pages and databases) without managing API keys. But the interface is too verbose for practical agent use. Each MCP tool call produces large markdown responses that consume context window, and chaining operations requires multiple round-trips through the LLM's neural network just to pass one tool's output to the next.

Traditional MCP tool-calling is inefficient: the LLM calls a tool, reads the full response, calls the next tool, reads again. Every intermediate result passes through the model, wasting tokens, time, and energy.

## The Inspiration

Cloudflare's [Code Mode](https://blog.cloudflare.com/code-mode/) proposes converting MCP tools into typed APIs so agents write code instead of making tool calls. The code executes in a sandbox with bindings to the MCP server; no API keys, no round-trips.

We take this further: **the agent doesn't need a custom sandbox. It already has one: bash.**

## The Insight

Bash is the original tool-chaining language. Pipes, `jq`, redirects, `grep`, `xargs`, subshells - these compose better than any SDK. AI agents are already fluent in shell commands. Instead of building a TypeScript runtime with MCP bindings, we build a CLI that speaks MCP underneath and returns compact output. The Unix toolkit does the rest.

```bash
# search and grab the first result
nt search "Q1 goals" | jq -r '.[0].id'

# pipe into read
nt page $(nt search "Q1 goals" | jq -r '.[0].id') read

# query a database with JSON filters
nt db abc123 query --filter '{"property":"Status","status":{"equals":"Done"}}' > done.json

# compose with any unix tool
nt search "meeting notes" | jq -r '.[].title' | grep -i "standup"

# copy content between pages
nt page <src> read | jq -r '.content' | nt page <dst> write -
```

No sandbox to build. No SDK to maintain. No new security model. The agent writes bash, and gets the full Unix ecosystem for free.

## Architecture

```
┌───────┐     HTTPS / MCP Streamable HTTP      ┌──────────────────┐
│  nt   │────────────────────────────────────→│  mcp.notion.com  │
│  CLI  │         stateless per-call           │  (Notion-hosted) │
└───────┘                                      └──────────────────┘
```

**`nt`** - A thin CLI binary. Parses arguments, sends an MCP JSON-RPC request over HTTPS to Notion's hosted MCP server, prints compact JSON, exits. Stateless: no daemon, no socket, no local MCP process.

**[Notion MCP](https://developers.notion.com/docs/mcp)** - Notion's hosted remote MCP server at `mcp.notion.com/mcp`. Supports Streamable HTTP transport natively. Handles authentication via OAuth and all Notion API communication. Actively developed by Notion, so we inherit improvements for free.

**Local state** - The only thing stored locally is the OAuth token (and refresh token) in `~/.config/nt/`. The CLI handles token refresh on 401. No other local state or configuration required for basic usage.

## Principles

**Wrap, don't reimplement.** The Notion MCP server already handles auth, API versioning, and Notion's quirks. We are a thin translation layer from CLI arguments to MCP tool calls, and from MCP responses to compact output. When the MCP server gains new capabilities, we gain them too.

**Compact by default.** Output is terse, structured JSON when piped, and human-readable when interactive (TTY). Every byte in the output should earn its place; agents pay for tokens, humans pay for attention.

**Composable over complete.** We don't need to cover every use case in the CLI itself. A small set of well-designed commands that compose with standard Unix tools is more powerful than a sprawling feature set. `nt search | jq | xargs nt fetch` should just work.

**Fast per-invocation.** The CLI is a compiled Go binary. Each invocation is a single HTTPS request to Notion's hosted MCP server; no local process startup, no connection pooling, no daemon. Commands complete in the time it takes for the Notion API to respond, with negligible local overhead.

**Progressive disclosure.** Simple things are simple (`nt search "meeting"`), complex things are possible (`nt query <db-id> --filter '{"property":"Status","status":{"equals":"Done"}}'`). Structured JSON for filters and sorts; no custom query syntax to learn or parse, and agents can generate it natively.

## Goals

1. **No API key required.** Authentication is handled via OAuth against Notion's hosted MCP server. A one-time `nt login` flow opens the browser, completes OAuth, and stores the token locally. No integration tokens, no manual configuration.

2. **Agent-first design.** The primary consumer is an AI agent running in a coding assistant (Claude Code, Codex, etc.). Output format, verbosity, and command structure should optimize for token efficiency and LLM parseability.

3. **Human-friendly second.** Interactive use should feel good (colored output, readable formatting, helpful errors), but never at the expense of agent ergonomics.

4. **Minimal surface area.** Start with the commands that cover 90% of agent workflows: search, fetch, create, update, query. Add more only when real usage demands it.

5. **Single binary distribution.** `go install` or a downloaded binary. No runtime dependencies, no package managers, no configuration files for basic usage.

## Why Go

- Zero startup cost: compiled binary, no runtime initialization tax
- Native HTTP client: `net/http` is battle-tested, no external dependencies needed
- Single binary: cross-compile for any platform, no dependencies
- Proven for CLIs: the same stack as `gh`, `kubectl`, `docker`, and the original `notion-cli`

## Prior Art

- **[notion-cli](https://github.com/4ier/notion-cli)** - Full Notion CLI in Go, talks directly to the Notion API. Excellent UX and command design. Requires an API integration token. Our reference for CLI ergonomics.
- **[Cloudflare Code Mode](https://blog.cloudflare.com/code-mode/)** - Converts MCP tools to TypeScript APIs for agent code generation. Inspired our thinking, but we chose bash composability over custom sandboxed code execution.
- **[Notion MCP (hosted)](https://developers.notion.com/docs/mcp)** - Notion's hosted remote MCP server with Streamable HTTP transport and OAuth. Our backend. Actively developed by Notion, replacing the older open-source local server.
