import { test, expect } from '@playwright/test';
import { createSatteriMarkdownProcessor, satteriHeadingIdsPlugin } from '@astrojs/markdown-satteri';
import { safeDocs, readingSurface, resolveDocLink } from '../src/docs/markdown.mjs';

test('canonical links preserve base, queries and fragments while hostile targets stay inert', async () => {
  for (const base of ['/kb/', '/']) {
    expect(resolveDocLink('quickstart.md?view=full#create-search-and-read', 'docs/README.md', base))
      .toBe(`${base}docs/quickstart/?view=full#create-search-and-read`);
    expect(resolveDocLink('../cli.md#exit-codes-and-errors', 'docs/qa/example.md', base)).toBe(`${base}docs/cli/#exit-codes-and-errors`);
    expect(resolveDocLink('qa/retrieval-quality.md#limits', 'docs/README.md', base))
      .toBe('https://github.com/matheus3301/kb/blob/main/docs/qa/retrieval-quality.md#limits');
    expect(resolveDocLink('#ação--api', 'docs/README.md', base)).toBe('#ação--api');
    for (const target of ['bad%xx.md', '%2e%2e/cli.md', '%2fetc/passwd', '//host/x', '/absolute', 'javascript:alert(1)', 'java%73cript:alert(1)', 'data:text/html,x', 'file:blocked', '%252e%252e/x', '..\\private.md', '../../../private.md']) {
      expect(resolveDocLink(target, 'docs/README.md', base), target).toBeNull();
    }
  }
});

test('native renderer drops executable HTML and unsafe links, preserves fenced content and GitHub heading identity', async ({ page }) => {
  const processor = await createSatteriMarkdownProcessor({
    features: { rawHtml: false }, mdastPlugins: [safeDocs],
    hastPlugins: [satteriHeadingIdsPlugin(), readingSurface],
  });
  const { code, metadata } = await processor.render('# Café, CLI & API!\n\n## Repeated\n\n## Repeated\n\n<script>window.compromised=true</script>\n\n[Unsafe](javascript:alert)\n\n[Malformed](bad%xx.md)\n\n![network](https://example.com/tracker.png)\n\n```html\n<script>example only</script>\n```', {
    fileURL: new URL('../../docs/README.md', import.meta.url),
  });
  expect(code).not.toMatch(/<script|<img|href="(?:javascript:|#")/);
  expect(code).toContain('Unsafe');
  expect(code).toContain('Malformed');
  await page.setContent(code);
  await expect(page.locator('pre code')).toHaveText('<script>example only</script>');
  await expect(page.locator('script, img')).toHaveCount(0);
  expect(metadata.headings.map(heading => heading.slug)).toEqual(['café-cli--api', 'repeated', 'repeated-1']);
});
