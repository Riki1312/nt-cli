# Context

Lightweight shared memory between humans and agents. Persists across sessions.

## Files

- `current.md` - Current state of active work. Read at the start of a run, update at the end.
- `decisions.md` - Append-only log of durable architectural and design decisions.
- `sessions/YYYY-MM-DD.md` - Optional notes for substantial work sessions.

## Rules

- Keep entries short and concrete.
- Timestamp with local date.
- Separate fact, hypothesis, and todo.
- Link evidence (file paths, commits, issues).
- Do not store secrets.
