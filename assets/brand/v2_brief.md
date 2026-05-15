# DumpSock v2 — Claude Recreation / Application Brief

This package contains the current **DumpSock v2** creative direction and a practical handoff for Claude Code / Claude Design.

## 1) What DumpSock is

DumpSock is a **light-theme, playful, cloudless backup product** for helping users free up storage on iPhone / iPad by backing up photos, videos, and files locally to a Mac, PC, Linux machine, external drive, or similar storage.

The tone should be:
- casual
- witty
- a little tongue-in-cheek
- useful first, joke second
- cute and memorable, not crude

## 2) Core creative direction

Claude should recreate / apply the design with these priorities:

1. **Mascot first**
   - The sock mascot is the core identity.
   - It should be a **straight hanging sock**, not angled or foot-shaped like a worn sock.
   - It should feel **cute, manga-adjacent, emoji-adjacent, and relatable**.
   - Facial expression should evoke the playful energy of the drooling emoji, but still be self-contained and tasteful.

2. **No external droplets**
   - The damp/wet idea must be implied **inside the sock shape**, especially around the mouth / toe-box area.
   - Do **not** add floating water droplets, sweat drops, splashes, puddles, or extra liquids outside the sock silhouette.

3. **Visual style**
   - Mostly white interface
   - Red highlights
   - Black / dark gray text
   - Bold red CTAs
   - Friendly rounded corners
   - Clean layouts with good whitespace
   - Minimal clutter

4. **Product meaning**
   - The joke is subtle.
   - The real value is clear: **local backup, privacy, storage recovery, cloudless workflows**.

## 3) Mascot rules

Required mascot traits:
- straight hanging sock
- ribbed cuff at the top
- 2 red stripe bands
- thick black outline
- white body with soft gray shading
- cute face centered in the upper-middle area
- drool/tongue vibe inside the mouth area only
- slightly damp / moist implication at the toe-box only
- self-contained silhouette

Avoid:
- realistic feet
- external water drops
- sweat beads around the mascot
- oversexualized rendering
- creepy/anatomical detailing
- dark cyberpunk treatment for this version

## 4) Brand tokens

### Colors
- Primary red: `#E62A28`
- Soft red: `#FF4444`
- Accent pink: `#FF8A8A`
- Ink black: `#111111`
- Charcoal: `#1F1F1F`
- Slate gray: `#4B5563`
- Rib gray: `#8E8E93`
- Sock shade: `#F2F2F4`
- White: `#FFFFFF`
- Drool blue: `#66D1FF`

### Fonts
- Logo / headings: **Space Grotesk**
- Body / UI: **Inter**
- Code / logs: **JetBrains Mono**

## 5) UI guidance by platform

### Mobile (iOS / Android)
- clean white cards
- big red primary action button
- obvious storage summary
- connected-device status
- progress UI for backup
- minimal bottom nav

### Desktop (macOS / Windows / Linux)
- left sidebar + main content area
- dashboard cards
- backup progress + compare/merge workflow
- clear destination path / storage device indicators
- light theme only for this version

### Web / Landing page
- product-first hero
- mascot used prominently
- CTA above the fold
- clear feature blocks:
     - local-first
     - frees up space
     - privacy
     - open / transparent / power-user-friendly

## 6) Files in this zip

### `/assets/reference_images/`
Reference mockups and style sheets generated during ideation.

### `/handoff/dumpsock_v2_brand_ui_handoff.html`
Main HTML handoff file. Claude can inspect this directly for:
- tokens
- layout intent
- SVG mascot structure
- icon style
- UI composition

### `/source_reference/`
Original direction supplied by the user.
Use these as grounding references for final mascot shape and vibe.

## 7) How Claude should use this package

Recommended order:

1. Read this markdown file.
2. Open and inspect `handoff/dumpsock_v2_brand_ui_handoff.html`.
3. Review the images in `assets/reference_images/` to understand the visual range.
4. Review the user-origin reference files in `source_reference/`.
5. Recreate final deliverables preserving the same identity system.

## 8) Expected deliverables Claude can build from this

Claude may use this package to produce:
- production HTML/CSS landing page
- React / Tailwind web implementation
- SwiftUI macOS app mockups
- Jetpack Compose Android screens
- iOS UIKit / SwiftUI screens
- SVG mascot/logo pack
- app icons / favicons
- design tokens JSON / YAML
- component library

## 9) Non-negotiables

- Keep the mascot hanging straight.
- Keep the latest playful light-theme direction.
- Keep the dampness implied internally only.
- Keep the product message practical and useful.
- Do not drift back into the earlier cyberpunk / dark / neon direction unless explicitly asked.

## 10) Suggested one-line product framing

> DumpSock is a playful, local-first tool that helps people free space on iPhone and iPad by backing up media and files the good old way: cloudless.


## 11) Correction note

The earlier HTML handoff contained a reconstructed mascot that did **not** sufficiently match the agreed design.

For all future work, treat these files as the **approved visual authority**:
- `assets/approved_assets/dumpsock_mascot_primary.png`
- `assets/approved_assets/dumpsock_horizontal_logo.png`
- `assets/approved_assets/dumpsock_stacked_logo.png`
- `assets/approved_assets/dumpsock_monochrome_logo.png`
- `assets/approved_assets/dumpsock_app_icons.png`
- `assets/reference_images/sock_mascot_brand_identity_sheet.png`

Use `handoff/dumpsock_v2_brand_ui_handoff_corrected.html` instead of the earlier HTML as the primary reference file.
