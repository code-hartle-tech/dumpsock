# Runbook — Cargo-installed tools

Tools we use during DumpSock development that come from `cargo install`.

## Why Rust tooling here

We aren't writing Rust for DumpSock. But Rust's tool ecosystem has some best-in-class CLI utilities that are easier to install via `cargo install` than to hunt down via Homebrew formulae.

## Prerequisites

```bash
# Install rustup if you don't have it
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source "$HOME/.cargo/env"
```

## vtracer

The reason this page exists.

```bash
cargo install vtracer
```

### When we use it

Whenever the mascot SVG needs regenerating from the master PNG. **Always**. Never redraw the SVG by hand. See [VTracer for raster→SVG](../discoveries/vtracer-for-raster-to-svg) for the full lesson.

### Recipe

```bash
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

# Copy into frontend + docs
cp assets/brand/dumpsock-mascot.svg \
   internal/frontend/dist/dumpsock-mascot.svg
cp assets/brand/dumpsock-mascot.svg \
   docs/external/public/dumpsock-mascot.svg
cp assets/brand/dumpsock-mascot.svg \
   docs/internal/public/dumpsock-mascot.svg
```

The flags are tuned for the current mascot. If we ever change the master art, tune them again — `vtracer` is interactive enough that a `--mode polygon` vs `--mode spline` switch can drastically change output size.

## tokei (optional)

Line counts for the README "by the numbers" badge.

```bash
cargo install tokei
tokei .
```

## fd (optional)

Faster `find`. Used in ad-hoc scripts only.

```bash
cargo install fd-find
```

## ripgrep (optional)

If `rg` isn't already on your box.

```bash
cargo install ripgrep
```

## Why not Homebrew?

For each of these, `brew install` exists too. We list `cargo install` paths because the operator's box has rustup already, and several of these (vtracer especially) lag behind on Homebrew or aren't bottled for `arm64`. `cargo install` always works on any platform with rustup.

## What we are NOT going to do

- Pull in any Rust crate as a dependency of the Go binary. CGo for Wails is already at our complexity budget.
- Run any of these tools in CI. They are dev-time only.
