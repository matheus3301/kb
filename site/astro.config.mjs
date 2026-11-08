import { defineConfig } from 'astro/config';
import { satteri, satteriHeadingIdsPlugin } from '@astrojs/markdown-satteri';
import { safeDocs, readingSurface } from './src/docs/markdown.mjs';

export default defineConfig({
  output: 'static',
  base: process.env.KB_SITE_BASE ?? '/kb/',
  trailingSlash: 'always',
  markdown: {
    shikiConfig: {
      theme: {
        name: 'kb-midnight',
        type: 'dark',
        colors: { 'editor.background': '#102940', 'editor.foreground': '#ecf3fb' },
        tokenColors: [
          { scope: ['comment', 'punctuation.definition.comment'], settings: { foreground: '#a4b5c9' } },
          { scope: ['string', 'keyword', 'constant', 'entity.name'], settings: { foreground: '#a8c7ff' } },
        ],
      },
    },
    processor: satteri({
      features: { rawHtml: false },
      mdastPlugins: [safeDocs],
      hastPlugins: [satteriHeadingIdsPlugin(), readingSurface],
    }),
  },
});
