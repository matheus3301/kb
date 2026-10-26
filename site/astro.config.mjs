import { defineConfig } from 'astro/config';

export default defineConfig({
  output: 'static',
  base: process.env.KB_SITE_BASE ?? '/kb/',
  trailingSlash: 'always',
});
