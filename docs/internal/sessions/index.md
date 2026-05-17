# Sessions

A running narrative of what we did, session by session. Future-Claude can resume from cold by reading the latest entry.

## Conventions

- One markdown file per session burst, named `YYYY-MM-{topic}.md`.
- Lead with the date and the close-state (e.g. "Closed at: `develop@abc1234`; tests green").
- Bullet-list the deliverables.
- Capture any decisions that aren't visible from the code.
- Link to the GitHub issues that were touched.

## Catalog

| Session | Window | Theme | Status |
|---|---|---|---|
| [2026-05 — kickoff → v3](./2026-05-kickoff) | 2026-05-13 → 2026-05-15 | Built the thing from scratch. CLI MVP. Wails GUI. Brand v3. Internal + external wikis. | Shipped |
| [2026-05-17 — phase 2.5 sweep](./2026-05-17-phase-2.5-sweep) | 2026-05-17 (single day) | Operator's 10-item GUI sweep + nonnegotiable workflow rules. Browse iPhone tab, third-party app data, byte progress, session restore, backups list, encrypted archives, scope expansion for Phase 6/7. | In progress — 14 of 21 tasks validated; 6 in-flight; docs sweep current |

## Why we keep these

The Claude conversation window compacts. Memory entries cover *what's still true*, not *what happened*. The sessions log is the only place we capture *what we did and why*, in temporal order. Reading the latest session before resuming work is the cheapest possible context-restore.
