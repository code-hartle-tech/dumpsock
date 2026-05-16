# VTracer for raster→SVG

## TL;DR

After two rounds of agents and three rounds of hand-authoring the DumpSock mascot in SVG, the only thing that produced a pixel-faithful result was **VTracer**, an MIT-licensed Rust CLI installed via `cargo install vtracer`. Hand-eyeballed bezier paths cannot reproduce what the source pixels actually say.

## What I tried that didn't work

### Round 1: Hand-authored SVG, attempt 1

Looked at the approved PNG, hand-coded `<path d="...">` data using my best guess at where the curves should go. Result: looked like a different sock. The proportions were off and the eyes were dead.

### Round 2: Hand-authored SVG, attempt 2

Tried again with `<defs>` for reusable curves, better proportions, hand-tuned bezier control points. Result, per the operator: "looks like ass still."

### Round 3: Online tracer (vectorizer.ai)

vectorizer.ai produces a beautiful trace. But the SVG path data is paywalled — you can preview but not export without a paid plan. Operator: "spin more agents and find something oss/libre".

### Round 4: 3 parallel research agents

Tested every credible OSS / libre / free-tier option:

| Tool | Verdict |
|---|---|
| **VTracer** (Rust CLI, MIT) | ✅ Best result. Local. Free. Tunable. |
| **Potrace** (C, free) | Monochrome only. We have a multi-color mascot. |
| **Autotrace** (C, GPL) | Old. Worse results than VTracer. |
| **Inkscape `Trace Bitmap`** | OK; tunability worse than VTracer. |
| svgco.de | Web tool. Works. Slower than local VTracer. |
| HF Space `ovi054/image-to-vector` | Web tool. Works. Slow. Same engine as svgco.de effectively. |

Conclusion: install VTracer. Use it for any mascot refresh.

## What worked

```bash
cargo install vtracer

vtracer \
  --input assets/brand/dumpsock_mascot_primary.png \
  --output assets/brand/dumpsock-mascot.svg \
  --colormode color \
  --mode polygon \
  --filter_speckle 4 \
  --color_precision 6 \
  --layer_difference 16 \
  --corner_threshold 60 \
  --segment_length 4 \
  --splice_threshold 45 \
  --path_precision 3
```

Result: 37 KB SVG that visually matches the PNG bit-for-bit at every viewport size we care about. The operator approved it on first look.

## Tuning notes

- `--colormode color` — keep all colors. (Default is binary; we don't want that.)
- `--mode polygon` — emit polygons; gives smaller files than spline mode for cartoon-style art.
- `--filter_speckle 4` — drop noise particles smaller than 4 px². Good for cleaning PNG JPEG-y artifacts.
- `--color_precision 6` — 6-bit color (64 levels per channel). Cuts file size with no visible loss for a mascot palette.
- `--layer_difference 16` — minimum hue gap between palette layers. Tighter = more layers = bigger file.
- `--corner_threshold 60` — degrees; below this, prefer smooth curves over corners.
- `--segment_length 4` — sub-divide longer paths for smoother curves.
- `--splice_threshold 45` — degrees; controls when adjacent segments merge.
- `--path_precision 3` — coordinate decimal places. 3 is usually enough; smaller = uglier curves.

## The mental model

> You cannot eyeball beziers and match what 200,000 source pixels actually say. The pixels carry information about edge density, color gradients, and fine-detail that a human can't enumerate by hand. Let a tracer do it.

> If you must hand-author SVG, do it for *new* art that has no raster reference. Never for a "redraw this PNG in SVG" task.

## Where the SVG ends up

- `assets/brand/dumpsock-mascot.svg` — canonical.
- `internal/frontend/dist/dumpsock-mascot.svg` — embedded in the GUI.
- `docs/external/public/dumpsock-mascot.svg` — public docs site.
- `docs/internal/public/dumpsock-mascot.svg` — this wiki.

Re-copy after any regen:

```bash
SRC=assets/brand/dumpsock-mascot.svg
/bin/cp -f $SRC internal/frontend/dist/
/bin/cp -f $SRC docs/external/public/
/bin/cp -f $SRC docs/internal/public/
```

(Use `/bin/cp` not `cp` — the operator's zsh aliases `cp` to a prompting variant that asks for overwrite confirmation, defeating the `-f`.)

## What we did NOT need

- A paid tracer license (vectorizer.ai is good but doesn't release the path data without payment).
- A web-only tool (worse for repeatability).
- An AI tracer (we tried; results inconsistent across runs).
- A vector illustration tool (Figma, Illustrator). They want you to draw, not trace, and the tracing tools they include are weaker than VTracer.

## Code pointers

- `scripts/build-icons.sh` consumes the PNG, not the SVG, to produce platform `.icns`/`.ico`.
- The SVG is only used by the GUI webview and the docs sites.
