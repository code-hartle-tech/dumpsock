---
title: DumpSock — Internal Wiki
---

# 🧦 DumpSock — Internal Wiki

This is the **inside-baseball** half of the DumpSock docs. Everything in here is fair game for naming internal hostnames, in-progress issues, lessons learned during dev, decisions made in chat that didn't make it into the code, and so on. The **public** half lives at [`docs/external/`](../external/) (or once deployed, [dumpsock.hartle.tech](https://dumpsock.hartle.tech)).

## Why two wikis?

External docs follow the HARTLE.TECH public voice: third-person, contact via `contact@hartle.tech`, no operator name, no internal hostnames, no in-progress work surfaced. Internal docs are the inverse — written for future-Claude and future-operator picking the project up cold, with full context.

Splitting them lets us write candidly here without having to filter every paragraph through the "is this safe to publish" lens. When something here graduates to public-ready, it gets adapted and pushed to `docs/external/`.

## Where to start

- **You're new to the codebase** → [Architecture](./dev/architecture)
- **You need to ship something** → [Cut a release](./runbooks/release)
- **You want to know how we did X** → [Discoveries](./discoveries/)
- **You want session history** → [Sessions](./sessions/)
- **You want to know the rules** → [Repo conventions](./dev/repo-conventions), [Brand bible](./dev/brand-bible)
- **GitHub board** → [Project 7 on `code-hartle-tech`](https://github.com/orgs/code-hartle-tech/projects/7)

## Operator handoff

If you are a future Claude session resuming this work, your read order is:

1. `MEMORY.md` (auto-loaded).
2. `CLAUDE.md` at repo root.
3. `SESSION_RESUME.md` if present.
4. [This wiki's sessions page](./sessions/) for the most recent close.
5. [Architecture](./dev/architecture) for the lay of the land.

Then verify state with `git status && git remote -v && git branch --show-current`.

## Voice for this wiki

- First-person plural ("we", "us") when the operator+Claude pair is meant.
- Honest about what's done, half-done, hacked-together, or skipped.
- Include `WHY` (motivation) more than `WHAT` (the code already says what).
- Snapshot timestamps when discussing in-flight state (sidebars and dashboards lie about being current).
- Casual but not sloppy. The reader is probably us, six months from now, and we will be grateful for context, not for jokes.

## Hosting

Build with `npm run internal:build` from `docs/`. Output lands in `docs/internal/.vitepress/dist/`. Intended deploy target: Tailnet-only Caddy under HARTLE.TECH's existing void.* convention. Local dev: `npm run internal:dev` → http://localhost:5175.

The `head` block sets `robots: noindex,nofollow` defensively, in case this ever gets accidentally pushed to a public host.
