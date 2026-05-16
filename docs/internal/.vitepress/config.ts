import { defineConfig } from 'vitepress'

// Internal DumpSock wiki. Served at https://dumpsock.hartle.tech/wiki/
// behind Caddy on the VPS — Caddy enforces tailnet-only access for this
// path (remote_ip matcher on 100.64.0.0/10). Public internet hits /wiki/
// gets a 404. Looks identical to the public site by design (same brand
// theme), just hosts different content.
//
// Run locally with `npm run internal:dev` (port 5175).

export default defineConfig({
  title: 'DumpSock — Wiki',
  description: 'Dev logs, runbooks, discoveries. Tailnet-only.',
  lang: 'en-US',
  cleanUrls: true,
  // Served at https://dumpsock.hartle.tech/wiki/ via Caddy on the VPS.
  // Tailnet-only: Caddy's @tailnet remote_ip 100.x matcher gates the
  // /wiki/* path. Tailscale extraDNSRecords (in tailscale_acl.tf)
  // overrides dumpsock.hartle.tech to the VPS tailnet IP for tailnet
  // members, so traffic reaches Caddy with a tailnet source IP.
  base: '/wiki/',
  // Brand spec (v2_brief.md §11): light-only. Same as external config.
  appearance: false,

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
    ['meta', { name: 'theme-color', content: '#E62A28' }],
    // Access control is enforced by Caddy at the network layer. noindex
    // is belt-and-suspenders so any tailnet member who shares a link
    // doesn't accidentally see Google crawl it.
    ['meta', { name: 'robots', content: 'noindex,nofollow' }],
  ],

  themeConfig: {
    logo: { src: '/dumpsock-mascot.svg', alt: 'DumpSock' },
    siteTitle: 'DumpSock — Wiki',

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
      message: 'Internal · Tailnet-only · <a href="/">back to the public site</a>',
      copyright: '© HARTLE.TECH',
    },

    search: { provider: 'local' },
  },
})
