# Comments and discussions

Scope for nt v0.3.0. MCP request shapes were checked against the authenticated hosted tools. This document records the feature boundary; [cli-design.md](../cli-design.md#comments-and-discussions) owns the implemented command and output contract.

## Purpose and boundary

Make comments, existing discussions, and reactions usable from shell scripts and agents. nt remains a stateless OAuth client. Callers own notifications, requester authorization, sessions, deduplication, and workflow decisions.

## Commands

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
nt tools TOOL_NAME
```

A discussion URL contains page, block, and discussion identifiers. Infer the page from that URL; `--page` is only valid for a bare discussion ID. Reject conflicting targets and irrelevant flags before authentication or network calls. Text arguments accept `-` for stdin and preserve their UTF-8 content, with a 1 MiB client limit. Markdown content uses the provider's supported inline formatting and mention syntax.

`--select` passes Notion's native `selection_with_ellipsis` selector. A unique match within one block creates an inline highlight; otherwise Notion can attach the comment to the block where the match starts. No match fails. The CLI adds no fuzzy matching or automatic page-level fallback. Live acceptance verified an inline anchor on a unique passage.

Reactions set the connected user's state using `notion-set-comment-reaction`: `reacted=true` to add, `false` to remove. Repeating the same state is idempotent. Provider plan/capability restrictions must remain visible; current documentation restricts these MCP controls to Business/Enterprise. The available test workspace accepted these operations without a plan change. Reactions do not resolve discussions.

## Output and completeness

Replace the opaque comment XML string with JSON records for discussions and comments. Preserve provider IDs/URLs, available author information, self markers, timestamps, anchors, and resolution state. Reactions have their own explicit state response; comment listings do not currently expose them. Do not infer identities or fabricate missing metadata.

Expose `shown-count`, `total-count`, omitted discussion URLs, `omitted-truncated`, per-discussion comment counts, and `next-cursor`. Fetch omitted discussions explicitly; use a discussion cursor to retrieve older comments. Do not silently report a partial result as complete or automatically fetch an unbounded history.

Comment creation and replies return the actual comment and discussion identifiers. Keep `--raw` as the original MCP result. Reject malformed/unsupported responses rather than returning an empty list or claiming an unverified write succeeded. Derive synthetic fixtures from observed response shapes; never commit real workspace content.

## Ownership and failures

CLI handlers validate inputs and shape MCP requests. Transform code owns response normalization. The MCP client owns bounded discovery and calls. Existing OAuth login remains unchanged.

Bound MCP calls to one minute, respect earlier cancellation, and do not replay uncertain writes. A lost response or unreadable successful result may follow a completed comment: inspect the discussion before another attempt. Return provider errors as nonzero JSON errors, including permission, rate-limit, and plan restrictions.

## Delivery

1. Add schema inspection, verify live tools, and review this design.
2. Implement structured reads, selections, discussion replies, and reactions in focused slices. Review touched code, names, tests, help, and docs after each slice.
3. Cover targeting, continuation/completeness, malformed responses, stdin, Unicode, capability failures, and uncertain writes. Run formatting, vet, race tests, minimum-Go checks, and builds; add CI that gates release.
4. Test on an agreed scratch Notion page, then publish v0.3.0 with GoReleaser and verify binaries/Homebrew. Deployment to consuming applications is separate.

Comment JSON output changes in this minor pre-1.0 release and must be called out in release notes. No automatic Notion conversation is enabled by publishing nt.
