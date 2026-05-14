# SESSION_RESUME — DumpSock

> One-paragraph close-of-session snapshot, refreshed every session. Goal: next session rebuilds full state in under a minute.

**Last updated:** 2026-05-14

---

## Where we are

**Phase 0 — scaffold.** The per-project skeleton mandated by https://void.neartrace.app/claude/handoff has been laid down:

- `CLAUDE.md` (house rules: stack, voice, icon spec, CLI surface, phase plan, anti-patterns)
- `.claude/settings.local.json` (Go tooling allow-list)
- Memory dir at `~/.claude/projects/-Users-vz-Projects-dumpsock/memory/`
- This file

Git repo initialized locally on `develop`. **No `gh repo create` yet** — awaiting operator confirm on:

1. **Repo name** — `dumpsock` (clean) vs `dumpsock-cli` (CLI-scoped, leaves room for a separate GUI repo) vs other.
2. **Project board** — new (#7) for DumpSock, or fold under existing #6 DevOps/Infrastructure.
3. **Domain / subdomain** — `dumpsock.app` (own registrable, brand-clean) vs `dumpsock.hartle.tech` (subdomain, cheaper) vs defer until Phase 2.

## What's running

- Background process `bb486wtjk` is `iphonepd.py` pulling the operator's 74 GB iPhone library to `/Volumes/Lexar/Backup/iCloud/Photos/` per icloudpd's `YYYY-MM-DD/` paradigm. Log at `/Volumes/Lexar/Backup/iphonepd.log`. Independent of DumpSock work.

## Next step on resume

1. Operator confirms the three open decisions above (`gh` action requires it — visible-to-others state).
2. `gh repo create code-hartle-tech/<name> --private --confirm` (private until Phase 2 / Phase 3 — public release later).
3. `git push -u origin develop`.
4. Create board / first issue (`chore(scaffold): phase 0 — repo skeleton + brand bible #1`).
5. Branch `feature/cli-skeleton` off `develop`.
6. `go mod init github.com/code-hartle-tech/<name>`.
7. Cobra-based CLI skeleton (`dumpsock`, `dumpsock pull`, `dumpsock devices`, `dumpsock version`).
8. Port `iphonepd.py` to Go using `github.com/danielpaulus/go-ios`.

## Open questions for operator (carry over until answered)

- Repo name?
- Project board: new or reuse?
- Subdomain / registrable domain decision?
- Any preference between Cobra and a hand-rolled flag parser? (Cobra is the default per `CLAUDE.md`; flagging in case operator wants stricter "no dependencies".)
- Confirm MIT license? (Matches neartrace-mvp convention — flagging because I haven't read those LICENSE files.)
