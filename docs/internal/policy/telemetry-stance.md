# Telemetry stance

## The rule

DumpSock **does not collect telemetry, ever**. Not opt-in. Not anonymized. Not "just crash reports". Nothing.

If a future contributor proposes adding telemetry — even opt-in, even anonymized, even for the noblest reason — the proposal needs an explicit operator approval **and** a public design doc **and** a separate release announcement before merging. There is no fast-path.

## Why this is non-negotiable

1. **Marketing positioning.** DumpSock's entire pitch is "the bytes go phone-to-disk, nowhere else". The instant we add a single telemetry beacon, that pitch becomes a lie.
2. **Threat-model alignment.** Some of our users care about local backup specifically because they don't trust cloud. They picked DumpSock to avoid network round-trips for their photos. A network round-trip for telemetry violates their choice.
3. **Maintenance cost.** Telemetry endpoints need uptime, security, GDPR/CCPA paperwork, breach response plans, and lawyer review. None of which we want.
4. **Org policy.** HARTLE.TECH's general posture is "products are tools, not surveillance"; telemetry contradicts that posture.

## What we DO use, that touches the network

1. **Google Fonts** for SF Pro / Inter / JetBrains Mono on first GUI open. Cached after that. The font request goes to `fonts.googleapis.com` and `fonts.gstatic.com`. We documented this in [public Privacy features](../../external/features/privacy).
   - **Mitigation available**: a one-line patch to `frontend/dist/index.html` vendors the fonts locally. Documented for users who want full air-gap.
   - **Why not vendor by default**: vendor fonts are ~3 MB extra in the binary, and 99% of users get the CDN cache hit.
2. **GitHub Releases** for manual update check. Not implemented yet. When implemented, it will be an explicit user-triggered button only.

That's it. Everything else (the backup itself, the AFC traffic, the config persistence) is local I/O.

## What we will NEVER do

- Send crash reports. (Crashes are debugged from logs the user shares, not from auto-uploads.)
- Send usage analytics. (Adoption is measured from GitHub Release download counts, not from in-product pings.)
- Send error logs. (See above.)
- Send "have you seen our other products" promos.
- Embed third-party SDKs that phone home (Firebase, Sentry, Crashlytics, Amplitude, etc.).

## What if a user explicitly asks for a crash report upload?

They can run `dumpsock pull --json | tee dumpsock.log` and email `dumpsock.log` to `contact@hartle.tech`. That's the support path. It's manual, explicit, and the user sees what they're sending.

## What about Update notifications?

Eventually, the app should be able to say "v0.3 is out, want to download it?" without uploading anything itself. The implementation: a static `latest.json` file in our GitHub Releases. The CLI / GUI fetches it on user click (NOT auto). Compares versions. Shows a banner if new. No identifying info is sent to GitHub other than the IP doing the fetch — which GitHub logs regardless of whether we ask.

Mark this as "explicit user click only" in any future implementation.

## How we enforce this

- **Code review:** any PR that adds a `net/http` Client or `Dial`-equivalent for outbound traffic must be flagged and discussed.
- **CI lint:** TODO — add a `go:linkname` audit or grep-based CI check that fails on outbound network calls in the binary, with a whitelist for `lockdownd` (local Unix socket) and `usbmuxd` (also local).
- **Documentation:** this page is the canonical reference. Any future telemetry discussion gets linked here as the "previously decided" anchor.

## How users can verify

- `Little Snitch` on macOS shows every outbound connection. DumpSock should show only `fonts.googleapis.com` + `fonts.gstatic.com` on first open and nothing thereafter.
- `lsof -i -P -n -p $(pgrep DumpSock)` lists open sockets.
- The source code is public (will be MIT once we hit v1.0); the audit is feasible.

## Related

- [public Privacy](../../external/features/privacy) — what we tell users.
- [public Privacy policy](../../external/privacy) — the legal-ish version.
- [Anonymity](./anonymity) — the operator-identity counterpart.
