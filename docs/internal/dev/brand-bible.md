# Brand bible

The single source of truth for the DumpSock visual + verbal identity. If `assets/brand/brand-bible.md` ever diverges from this page, that one wins — but they should track.

## The mascot

Approved primary mascot is `assets/brand/dumpsock_mascot_primary.png`. SVG counterpart is `assets/brand/dumpsock-mascot.svg`, generated via [VTracer](../runbooks/cargo-tools#vtracer) — never hand-authored.

Hard rule: **never redraw the mascot by hand.** Eyeballed beziers look like ass. If the SVG needs a refresh, run VTracer again from the PNG. See [VTracer for raster→SVG](../discoveries/vtracer-for-raster-to-svg) for the full lesson.

The mascot is a damp-looking sock. The vibe is "exhausted but cheerful", evoking a sock that's just been wrung out — which is exactly what we're metaphorically doing to the iPhone (wringing the photos out of it).

### Where it appears

- macOS `.app` icon (built via `scripts/build-icons.sh` → `sips` + `iconutil`).
- Windows `.ico` and Linux `.png` (`icotool`).
- App sidebar header.
- About modal.
- Both VitePress wikis' favicon + logo.
- Repo README banner.
- GitHub social preview.

### Where it does NOT appear

- Receipts, emails, or any formal HARTLE.TECH-org communication. The org identity stays org-level; the mascot is product-level.

## Color tokens (v3, current)

| Token | Hex | Use |
|---|---|---|
| `--brand-primary` | `#E74C3C` | Primary CTA, active sidebar pill, accent strokes |
| `--brand-primary-bg` | `#FEECEE` | Selected-row backgrounds, primary-button hover tint |
| `--brand-primary-ink` | `#A93226` | Active text inside primary CTAs (darker for AA contrast) |
| `--ink` | `#111111` | Body text |
| `--ink-2` | `#5A6270` | Secondary text |
| `--ink-3` | `#8A93A1` | Tertiary / placeholder |
| `--subtle` | `#F7F8FA` | Window background, card backgrounds |
| `--surface` | `#FFFFFF` | Elevated card |
| `--line` | `#E6E8EE` | Hairlines, dividers |
| `--ok` | `#22C55E` | Success states, "connected" dot |
| `--warn` | `#F59E0B` | Warning toasts, indeterminate progress |
| `--danger` | `#EF4444` | Destructive actions, errors |
| `--info` | `#3B82F6` | Informational toasts, neutral status |

### Color v1 / v2 → v3 migration

We had `#E62A28` as primary briefly (a stricter red). Operator's v3 zoom-in mockups specified `#E74C3C` (a touch warmer). Switched on first v3 pass. Do not regress.

## Typography

| Use | Family | Size | Weight |
|---|---|---|---|
| H1 | SF Pro / Inter | 40 px / 1.1 | 700 |
| H2 | SF Pro / Inter | 28 px / 1.15 | 600 |
| H3 | SF Pro / Inter | 20 px / 1.2 | 600 |
| Body | SF Pro / Inter | 14 px / 1.5 | 400 |
| Caption | SF Pro / Inter | 12 px / 1.4 | 500 |
| Code / numerics | JetBrains Mono | inherit | 500 |

`SF Pro` on macOS via system font; `Inter` fallback elsewhere. Both loaded once on first frame from Google Fonts (acceptable trade-off; see [Telemetry stance](../policy/telemetry-stance)).

## Spacing scale

Multiples of 4: `4, 8, 12, 16, 20, 24, 32, 40, 48, 64`. Don't introduce odd values. The sidebar nav row is 44 px tall; cards have 24 px padding; the chrome bar is 36 px.

## Geometry

| Element | Spec |
|---|---|
| Window corner radius | 14 px (Wails handles via `--webkit-app-region`) |
| Card radius | 12 px |
| Button radius | 10 px |
| Pill / badge radius | 999 px |
| Active sidebar item radius | 12 px |
| Sidebar width | 240 px |
| Sidebar item height | 44 px |
| Sidebar icon size | 24 px |
| Chrome bar height | 36 px |
| Chrome bar left-pad | 88 px (macOS traffic lights live in the first ~80 px) |

## Animation grammar

| Token | Curve | Duration |
|---|---|---|
| `--ease-snap` | `cubic-bezier(.2, .8, .2, 1)` | 160 ms |
| `--ease-deep` | `cubic-bezier(.4, 0, .2, 1)` | 280 ms |
| `--ease-bounce` | `cubic-bezier(.34, 1.56, .64, 1)` | 420 ms |

Named keyframes in `style.css`:
- `shimmer` — indeterminate progress bars.
- `pulse-ok` — green status dot when connected and idle.
- `fadeIn` — modal entry, toast entry.
- `pop` — number-counter bumps on completion.

## Voice

External docs voice:
- Plain English. Avoid jargon when a normal word fits.
- Concrete numbers ("70 GB" not "a lot"). When uncertain, hedge ("often", "in our tests") rather than claim.
- No "we believe", "we think", or other epistemic padding. Say it or don't.
- Cheerful but not jokey. Mascot allows one ":)" per page, max.
- Active voice. Present tense. Second person ("You plug your iPhone in") for tutorials, third person ("DumpSock writes the file") for reference.

Internal docs voice:
- First person plural OK ("we discovered", "we tried").
- Honest. Half-finished things get marked half-finished. Skipped things get marked skipped.
- Timestamp anything that decays.

Never use:
- "blazing fast", "delight", "magical", "elegant" (vapor).
- "leverage" as a verb.
- "stakeholder" — the user is the stakeholder.

## Wordmark / type lockup

For headlines: bold weight, slight letter tightening (`letter-spacing: -0.01em`). The wordmark next to the mascot uses `font-weight: 700` and `font-size: 20px` in the sidebar header.

## Photography (none yet)

We don't have any. If we ever add product shots, they should:
- Show real hardware (USB-C cable, iPhone, Lexar SSD).
- Be shot on a neutral surface (wood or off-white).
- Avoid Apple-marketing-clean (no perfect specular highlights).
- Carry a hint of mess — a real desk, not a stage set.

The energy is "the tool you actually use", not "the tool you ship to enterprise procurement".
