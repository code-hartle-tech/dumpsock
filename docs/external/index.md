---
layout: home
title: DumpSock
hero:
  name: DumpSock
  text: Cloudless freedom for your iPhone.
  tagline: Plug your phone in. Wring it dry. Local-first photo and video backup from iPhone / iPad to disk. No iCloud, no account, no telemetry.
  image:
    src: /dumpsock-mascot.svg
    alt: DumpSock mascot
  actions:
    - theme: brand
      text: Get started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/code-hartle-tech/dumpsock

features:
  - icon: 🔒
    title: 100% Local
    details: Photos go from your phone to your disk and nowhere else. No cloud round-trip. No middleman. No account.
  - icon: 🗂
    title: Date-sorted output
    details: Every photo and video lands in a YYYY-MM-DD folder based on its real capture date — the same layout icloudpd produces, so the two tools are interchangeable.
  - icon: 🧽
    title: Free iPhone storage
    details: Optional --delete-after removes each file from /var/mobile/Media/DCIM/ once it's safely on disk. Storage freed immediately.
  - icon: 🪞
    title: Compare & merge
    details: Side-by-side diff between your iPhone and your destination folder. See what's new on the phone, what's new on disk, what's different.
  - icon: 🛡
    title: No telemetry. Ever.
    details: DumpSock doesn't phone home. No analytics SDK, no crash reporter, no usage pings. The source is open and the build is reproducible.
  - icon: 🐧
    title: Cross-platform
    details: macOS GUI today. Linux and Windows CLI build from the same Go source. Native binaries, single executable, no install dance.
---
