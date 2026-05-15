import { defineConfig } from 'vitepress'

// Internal DumpSock wiki. Intended for Tailnet-only deployment behind
// the operator's existing void.neartrace.app Caddy convention, OR run
// locally with `npm run internal:dev` (port 5175).
//
// Anything in here may name internal paths, hostnames, dev tooling,
// in-flight issues. Public-safe content lives under docs/external/.

export default defineConfig({
  title: '🧦 DumpSock — Internal Wiki',
  description: 'Dev logs, runbooks, discoveries, classified.',
  lang: 'en-US',
  cleanUrls: true,
  // Served at https://dumpsock.hartle.tech/wiki/ — same subdomain as the
  // external /docs/ build. Public-but-unindexed: noindex headers + robots.txt
  // deny under /wiki/. Content here must NOT contain secrets — Rule #1.
  base: '/wiki/',

  // The internal wiki cross-references the external (separate VitePress
  // build) site and a localhost dev server. Those aren't resolvable from
  // a build of *this* site alone, so we tolerate dead links rather than
  // fail the build on cross-site references.
  ignoreDeadLinks: [
    // Cross-site refs to docs/external/* — these resolve once the external
    // site is deployed (e.g. at dumpsock.hartle.tech). VitePress resolves
    // these relative to the current page's location, so we just match any
    // path that names "external/" anywhere in the link.
    /external\//,
    // Internal dev server links (http://localhost:5175 etc.).
    /^https?:\/\/localhost/,
  ],

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/dumpsock-mascot.svg' }],
    ['meta', { name: 'theme-color', content: '#E74C3C' }],
    ['meta', { name: 'robots', content: 'noindex,nofollow' }],
  ],

  themeConfig: {
    logo: { src: '/dumpsock-mascot.svg', alt: 'DumpSock' },
    siteTitle: '🧦 DumpSock — Internal',

    nav: [
      { text: 'Dev', link: '/dev/architecture' },
      { text: 'Runbooks', link: '/runbooks/release' },
      { text: 'Discoveries', link: '/discoveries/' },
      { text: 'Sessions', link: '/sessions/' },
      { text: 'GitHub board', link: 'https://github.com/orgs/code-hartle-tech/projects/7' },
    ],

    sidebar: {
      '/': [
        {
          text: '🚀 Start',
          items: [
            { text: 'About this wiki', link: '/' },
            { text: 'Repo conventions', link: '/dev/repo-conventions' },
            { text: 'Brand bible', link: '/dev/brand-bible' },
          ],
        },
        {
          text: '🛠 Dev',
          items: [
            { text: 'Architecture', link: '/dev/architecture' },
            { text: 'Build & release', link: '/dev/build-and-release' },
            { text: 'Testing', link: '/dev/testing' },
            { text: 'GUI structure', link: '/dev/gui-structure' },
            { text: 'AFC + PTP behavior', link: '/dev/afc-and-ptp' },
          ],
        },
        {
          text: '📓 Runbooks',
          items: [
            { text: 'Cut a release', link: '/runbooks/release' },
            { text: 'Hand-test the GUI', link: '/runbooks/gui-smoke-test' },
            { text: 'Resort misplaced files', link: '/runbooks/resort-misplaced' },
            { text: 'Cargo-install tools', link: '/runbooks/cargo-tools' },
          ],
        },
        {
          text: '🧪 Discoveries',
          items: [
            { text: 'Index', link: '/discoveries/' },
            { text: '8 agents on Recently Deleted', link: '/discoveries/recently-deleted-cannot-be-bypassed' },
            { text: 'exiftool -fast2 trap', link: '/discoveries/exiftool-fast2-trap' },
            { text: 'PTP delete does NOT free bytes', link: '/discoveries/ptp-delete-still-trashes' },
            { text: 'AFC delete leaves ghosts in Photos.app', link: '/discoveries/afc-delete-ghosts' },
            { text: 'VTracer for raster→SVG', link: '/discoveries/vtracer-for-raster-to-svg' },
          ],
        },
        {
          text: '🗂 Sessions',
          items: [
            { text: 'Index', link: '/sessions/' },
            { text: '2026-05 — kickoff → v3', link: '/sessions/2026-05-kickoff' },
          ],
        },
        {
          text: '⚖ Policy',
          items: [
            { text: 'Anonymity', link: '/policy/anonymity' },
            { text: 'Telemetry stance', link: '/policy/telemetry-stance' },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/code-hartle-tech/dumpsock' },
    ],

    footer: {
      message: 'Internal. Tailnet-only.',
      copyright: '© HARTLE.TECH',
    },

    search: { provider: 'local' },
  },
})
