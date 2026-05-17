---
layout: home
title: DumpSock
hero:
  name: DumpSock
  text: Cloudless freedom for your iPhone.
  tagline: DumpSock backs up your iPhone and iPad to your computer. No iCloud, no subscriptions, no account, no telemetry — just clean, dated archives on the disk you already own.
  image:
    src: /dumpsock-mascot.svg
    alt: DumpSock mascot — a friendly hanging sock
  actions:
    - theme: brand
      text: Download for macOS
      link: https://github.com/code-hartle-tech/dumpsock/releases
    - theme: alt
      text: All downloads
      link: /guide/install
    - theme: alt
      text: GitHub
      link: https://github.com/code-hartle-tech/dumpsock

features:
  - icon: 🔒
    title: 100% Local
    details: Photos go from your phone to your disk and nowhere else. No cloud round-trip. No middleman. No account required, ever.
  - icon: 🛡
    title: Private by default
    details: No analytics SDK, no crash reporter, no usage pings. The source is open, the build is reproducible, and the binary phones nobody.
  - icon: 🪶
    title: Fast & lightweight
    details: One native binary per platform. No Electron. Backs up a 256 GB iPhone in roughly the time it takes USB-C to move the bytes.
  - icon: 🧽
    title: Free iPhone storage
    details: Optional Wring-it-dry removes each file from your phone the moment it's safely on disk. Storage freed immediately, no Recently-Deleted purgatory.
  - icon: 🔍
    title: Browse before you back up
    details: Pick exactly what to copy — single files, entire folders, or even a third-party app's Documents. No more all-or-nothing.
  - icon: 🗝
    title: Encrypted archives
    details: Optional AES-256-GCM password protection on the produced zip. PBKDF2 with 200,000 iterations. Solid for moving backups across untrusted boundaries.
  - icon: ❤️‍🩹
    title: Easy restore
    details: Drag any folder back through Image Capture, Finder, or Photos.app — nothing about DumpSock locks you into DumpSock.
  - icon: 🐧
    title: Open source
    details: Apache 2.0. macOS GUI today; Linux and Windows CLI build from the same Go source. Fork it, ship it, give credit, sleep well.
---

<div class="ds-tagline-strip">
  <span>100% Local</span>
  <span class="dot"></span>
  <span>Open Source</span>
  <span class="dot"></span>
  <span>macOS · Windows · Linux</span>
  <span class="dot"></span>
  <span>No Cloud, No Account</span>
</div>

<h2 class="ds-section-title">Back up in 3 ridiculously easy steps</h2>
<p class="ds-section-subtitle">No technical knowledge required. No setup wizard. Plug in, hit one button, walk away.</p>

<div class="ds-steps">
  <div class="ds-step">
    <div class="ds-step__number">1</div>
    <h3>Plug your phone in</h3>
    <p>USB-C or Lightning. Trust the computer when iOS asks. DumpSock detects the device and shows you what's on it — number of photos, videos, screenshots, plus total size.</p>
  </div>
  <div class="ds-step">
    <div class="ds-step__number">2</div>
    <h3>Hit "Back Up Now"</h3>
    <p>Pick a destination folder. DumpSock writes a date-sorted tree (YYYY-MM-DD/) with the original capture date preserved. Live progress, no surprises.</p>
  </div>
  <div class="ds-step">
    <div class="ds-step__number">3</div>
    <h3>You're done</h3>
    <p>Optionally enable --delete-after to free space on the phone. Or don't. Either way, your files are on your disk, in folders you can browse without DumpSock ever running again.</p>
  </div>
</div>

<h2 class="ds-section-title">Loved by humans, trusted by data hoarders</h2>
<p class="ds-section-subtitle">Illustrative quotes — DumpSock is brand-new, so these are written in the spirit of the people we built it for. <a href="mailto:contact@hartle.tech">Send us yours</a> once you've used it for real.</p>

<div class="ds-testimonials">
  <div class="ds-testimonial">
    <p class="ds-testimonial__quote">"I have a NAS, three external drives, and zero photos in the cloud. DumpSock is the missing link between my iPhone and the disk I actually trust."</p>
    <div class="ds-testimonial__author">
      <span class="ds-testimonial__avatar">DH</span>
      <span>Data hoarder · Lisbon</span>
    </div>
  </div>
  <div class="ds-testimonial">
    <p class="ds-testimonial__quote">"My iCloud was 'almost full' for three years. DumpSock + --delete-after gave me back 180 GB on the phone in one evening, and the originals are on my own RAID where they belong."</p>
    <div class="ds-testimonial__author">
      <span class="ds-testimonial__avatar">CR</span>
      <span>Cloud refugee · anonymous</span>
    </div>
  </div>
  <div class="ds-testimonial">
    <p class="ds-testimonial__quote">"Cute mascot, honest software. No tracking, no upsell, no 'pro' tier. Donated because I wanted to, not because they paywalled the dark-mode toggle (which doesn't exist — and that's the point)."</p>
    <div class="ds-testimonial__author">
      <span class="ds-testimonial__avatar">OS</span>
      <span>Open-source supporter</span>
    </div>
  </div>
</div>

<h2 class="ds-section-title">Want to support DumpSock?</h2>
<p class="ds-section-subtitle">Donations are voluntary and never gate any feature. If DumpSock saved you an iCloud subscription or recovered storage you needed, you can drop us something below — no strings, no perks, just a thank you.</p>

<div style="text-align: center; margin: 2rem 0 4rem 0;">
  <a href="/donate" class="VPButton brand" style="display: inline-block;">See donation options</a>
</div>
