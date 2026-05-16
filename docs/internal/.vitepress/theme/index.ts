// Internal wiki theme — same brand tokens as the external site so /docs/
// and /wiki/ feel like one product. Only difference: the internal wiki
// loads an additional tiny stylesheet that surfaces a subtle "INTERNAL"
// banner across the top, so anyone landing there from a search/share knows
// the content shape is dev-facing notes, not polished marketing copy.

import DefaultTheme from 'vitepress/theme'
import './style.css'

export default DefaultTheme
