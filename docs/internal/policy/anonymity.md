# Anonymity policy

## The rule

Public-facing surfaces credit **HARTLE.TECH** and only HARTLE.TECH. The operator's name, handles, and personal identity do not appear in:

- GitHub commit authors (git config sets `hartle-tech <noreply@hartle.tech>`).
- GitHub PR/issue text.
- Release notes.
- The DumpSock GUI's About modal.
- External wiki pages.
- Any social or marketing surface.
- README files.
- Any binary artifact's metadata (`-X main.Author=` etc.).

## Why

1. **Threat surface reduction.** A product page that lists an individual's name is a phishing-pattern starter for that individual's friends/family.
2. **Org consistency.** HARTLE.TECH ships multiple things. They all credit the org. Mixing personal credits muddies the brand.
3. **Operator preference.** Personal preference for staying out of the public attribution chain.

## What's allowed publicly

- `HARTLE.TECH` as the org.
- `contact@hartle.tech` for inbound communication.
- The DumpSock mascot, name, wordmark, and product identity.
- The MIT license text once we declare it (a license file does not require a personal-name copyright holder — `Copyright (c) HARTLE.TECH` is sufficient).

## What's NOT allowed publicly

- The operator's name.
- The operator's email (other than `contact@hartle.tech`).
- The operator's handles (Twitter / GitHub / Mastodon / wherever).
- The operator's physical location (city / country).
- Personally identifying screenshots (a phone wallpaper with a family photo, a desktop screenshot with their name in the menu bar, etc.).

## What IS allowed internally

This wiki, marked `noindex,nofollow`, Tailnet-only, can reference the operator by handle if needed for clarity. Same for memory entries, runbooks, decision logs.

The split is: **internal can be candid; external must be HARTLE.TECH-clean.**

## Operational checklist

Before pushing anything to a public surface:

- [ ] `git log --pretty='%an <%ae>'` shows only `hartle-tech <noreply@hartle.tech>`.
- [ ] No screenshot leaks the macOS menu bar's user name. (Crop, or use a clean test user.)
- [ ] Wallpapers in screenshots are neutral. The default DumpSock screenshot wallpaper is a single solid color.
- [ ] Phone screenshots have the time / battery / wifi-name redacted.
- [ ] No references to the operator's other handles.
- [ ] `contact@hartle.tech` is the only contact route surfaced.

## If something leaks

1. **Don't panic, don't lie.** A leak does not retract by being denied.
2. Rewrite history if the leak is in commit messages or a recent push: amend or rebase, force-push (within the safety rules — never on `main`).
3. Issue a follow-up PR / commit that explicitly redacts.
4. If the leak is in a published binary or release asset, replace the asset and rotate any keys/credentials that could be inferred.
5. Note the incident in `discoveries/` so the failure mode is documented.

## Memory layer

The operator's identity is intentionally NOT stored in any memory file. References in this wiki to "the operator" mean exactly that — the human at the keyboard — and a Claude session resuming cold should NOT try to look up which human that is. That information lives outside this project.

## Related

- [Telemetry stance](./telemetry-stance) — for the same reasons, we don't collect anything that could re-attach an identity.
- HARTLE.TECH org-level contact policy lives in the org's main playbook, not this wiki.
