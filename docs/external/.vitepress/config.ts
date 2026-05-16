import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'DumpSock',
  description: 'Cloudless freedom for your iPhone. Local-first photo and video backup.',
  lang: 'en-US',
  cleanUrls: true,
  // Served at root of https://dumpsock.hartle.tech/ — the marketing landing
  // IS the site. Guide nav link routes users into the documentation pages
  // (/guide/, /features/, /reference/). Internal wiki remains at /wiki/.
  base: '/',
  // Brand spec (assets/brand/v2_brief.md §11): light-only. Disable the
  // VitePress dark-mode toggle entirely — half-suppressing it via CSS
  // creates a broken light-body/dark-cards state.
  appearance: false,

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/dumpsock-mascot.svg' }],
    ['link', { rel: 'apple-touch-icon', href: '/dumpsock-mascot.svg' }],
    ['link', { rel: 'mask-icon', href: '/dumpsock-mascot.svg', color: '#E62A28' }],
    ['meta', { name: 'theme-color', content: '#E62A28' }],
    ['meta', { name: 'color-scheme', content: 'light' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:url', content: 'https://dumpsock.hartle.tech/' }],
    ['meta', { property: 'og:title', content: 'DumpSock — Cloudless freedom for your iPhone' }],
    ['meta', { property: 'og:description', content: 'Plug your phone in. Wring it dry. Local-first photo and video backup from iPhone / iPad to disk. No iCloud, no account, no telemetry.' }],
    ['meta', { property: 'og:image', content: 'https://dumpsock.hartle.tech/dumpsock-mascot.svg' }],
    ['meta', { property: 'og:site_name', content: 'DumpSock' }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { name: 'twitter:title', content: 'DumpSock — Cloudless freedom for your iPhone' }],
    ['meta', { name: 'twitter:description', content: 'Local-first photo + video backup from iPhone/iPad. No cloud, no account.' }],
    ['meta', { name: 'twitter:image', content: 'https://dumpsock.hartle.tech/dumpsock-mascot.svg' }],
  ],

  themeConfig: {
    logo: { src: '/dumpsock-mascot.svg', alt: 'DumpSock' },
    siteTitle: 'DumpSock',

    nav: [
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Features', link: '/features/local-first' },
      { text: 'FAQ', link: '/faq' },
      { text: 'Donate', link: '/donate' },
      { text: 'Download', link: 'https://github.com/code-hartle-tech/dumpsock/releases' },
      { text: 'Source', link: 'https://github.com/code-hartle-tech/dumpsock' },
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'Get started',
          items: [
            { text: 'What is DumpSock?', link: '/guide/getting-started' },
            { text: 'Install', link: '/guide/install' },
            { text: 'Your first backup', link: '/guide/your-first-backup' },
            { text: 'Free up iPhone storage', link: '/guide/free-storage' },
            { text: 'CLI usage', link: '/guide/cli' },
          ],
        },
      ],
      '/features/': [
        {
          text: 'Features',
          items: [
            { text: 'Local-first', link: '/features/local-first' },
            { text: 'Privacy', link: '/features/privacy' },
            { text: 'Delete-after', link: '/features/delete-after' },
            { text: 'Compare & Merge', link: '/features/compare' },
            { text: 'Date-sorted output', link: '/features/date-sorted' },
          ],
        },
      ],
      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'CLI flags', link: '/reference/cli-flags' },
            { text: 'File layout on disk', link: '/reference/file-layout' },
            { text: 'Supported devices', link: '/reference/supported-devices' },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/code-hartle-tech/dumpsock' },
    ],

    footer: {
      message: 'A <a href="https://hartle.tech">HARTLE.TECH</a> tool · <a href="mailto:contact@hartle.tech">contact@hartle.tech</a>',
      copyright: '© HARTLE.TECH',
    },

    search: { provider: 'local' },
  },
})
