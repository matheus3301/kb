import AxeBuilder from '@axe-core/playwright';
import { expect, test as base, type Page } from '@playwright/test';
import { documents, docsPath } from '../src/docs/catalog.mjs';

const test = base.extend({
  page: async ({ page }, use) => {
    const errors: string[] = [];
    page.on('pageerror', error => errors.push(error.message));
    page.on('console', message => { if (message.type() === 'error') errors.push(message.text()); });
    page.on('requestfailed', request => errors.push(`${request.url()}: ${request.failure()?.errorText}`));
    page.on('response', response => { if (response.status() >= 400) errors.push(`${response.status()} ${response.url()}`); });
    await page.route('**/*', route => {
      const url = new URL(route.request().url());
      if (/^https?:$/.test(url.protocol) && url.origin !== 'http://127.0.0.1:4321') {
        errors.push(`External runtime request: ${url.href}`);
        return route.abort();
      }
      return route.continue();
    });
    await use(page);
    expect(errors).toEqual([]);
  },
});

async function accessible(page: Page) {
  const { violations } = await new AxeBuilder({ page }).analyze();
  expect(violations.filter(({ impact }) => impact === 'critical' || impact === 'serious')).toEqual([]);
}
async function noOverflow(page: Page) {
  expect(await page.evaluate(() => Math.max(document.body.scrollWidth, document.documentElement.scrollWidth) - innerWidth)).toBeLessThanOrEqual(1);
}

test('every published route reads correctly, links and all heading fragments resolve, wide material stays contained', async ({ page, baseURL }, info) => {
  test.setTimeout(120_000);
  const basePath = new URL(baseURL!).pathname;
  const ids = new Map<string, Set<string>>();
  const links: { source: string; url: string }[] = [];
  for (const doc of documents) {
    const route = docsPath(doc.id, basePath);
    await page.goto(route);
    await expect(page.locator('article h1')).toHaveCount(1);
    await expect(page.locator('article')).not.toContainText('generated:');
    await expect(page.locator('.docs-sidebar a[aria-current]')).toHaveAttribute('href', route);
    const content = await page.evaluate(() => ({
      ids: [...document.querySelectorAll('[id]')].map(node => node.id),
      links: [...document.querySelectorAll<HTMLAnchorElement>('a[href]')].map(a => a.href).filter(href => new URL(href).origin === location.origin),
      headings: [...document.querySelectorAll<HTMLElement>('article :is(h1,h2,h3,h4,h5,h6)')].map(heading => ({ id: heading.id, href: heading.querySelector('a')?.getAttribute('href') })),
      tables: [...document.querySelectorAll('article table')].every(table => table.parentElement?.classList.contains('table-scroll')),
    }));
    expect(new Set(content.ids).size, `Duplicate IDs in ${route}`).toBe(content.ids.length);
    for (const heading of content.headings) expect(heading.href).toBe(`#${heading.id}`);
    expect(content.tables).toBe(true);
    ids.set(route, new Set(content.ids));
    links.push(...content.links.map(url => ({ source: route, url })));
    await noOverflow(page);
    await accessible(page);
    if (doc.id === 'README' || doc.id === 'cli' || doc.id === 'setup') {
      await page.screenshot({ path: info.outputPath(`docs-${doc.id.toLowerCase()}-${info.project.name}.png`), fullPage: doc.id === 'README' });
    }
  }
  for (const { source, url } of links) {
    const target = new URL(url);
    if (target.pathname === basePath) continue;
    expect(ids.has(target.pathname), `Unpublished local link from ${source}: ${url}`).toBe(true);
    if (target.hash) expect(ids.get(target.pathname)!.has(decodeURIComponent(target.hash.slice(1))), `Missing fragment from ${source}: ${url}`).toBe(true);
  }
});

test('home and documentation navigation work by keyboard, with mobile menu Escape and visible focus', async ({ page, isMobile }, info) => {
  await page.goto('./');
  await page.getByRole('navigation', { name: 'Main navigation' }).getByRole('link', { name: 'Docs', exact: true }).focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('article h1')).toHaveText('Documentation');
  if (isMobile) {
    const summary = page.locator('.docs-menu summary');
    await summary.focus(); await summary.press('Enter');
    await expect(page.getByRole('navigation', { name: 'Documentation', exact: true })).toBeVisible();
    await page.screenshot({ path: info.outputPath('docs-menu-mobile.png') });
    await page.getByRole('navigation', { name: 'Documentation', exact: true }).getByRole('link', { name: 'Quickstart', exact: true }).focus();
    await page.keyboard.press('Escape');
    await expect(summary).toBeFocused();
    await expect(page.locator('.docs-menu')).not.toHaveAttribute('open', '');
    await summary.press('Enter');
  }
  const quickstart = page.getByRole('navigation', { name: 'Documentation', exact: true }).getByRole('link', { name: 'Quickstart', exact: true });
  await quickstart.focus();
  expect(await quickstart.evaluate(element => getComputedStyle(element).outlineStyle)).not.toBe('none');
  await quickstart.press('Enter');
  await expect(page.locator('article h1')).toHaveText('Quickstart');
  await page.locator('article h2 a').first().focus();
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(/#install-the-cli$/);
});

test('local search ranks titles, finds body content, handles keyboard selection, empty results, close and focus restoration', async ({ page }, info) => {
  await page.goto('docs/');
  const trigger = page.getByRole('button', { name: /Search docs/ });
  await trigger.focus();
  await page.keyboard.press('Control+k');
  const dialog = page.getByRole('dialog');
  const query = page.getByRole('searchbox');
  await expect(query).toBeFocused();
  await query.fill('setup');
  await expect(page.locator('#search-results a').first()).toHaveAttribute('href', /\/docs\/agent-setup\/$/);
  await query.fill('kb');
  await expect(dialog.getByRole('status')).toContainText('Showing');
  const counts = (await dialog.getByRole('status').innerText()).match(/Showing (\d+) of (\d+)/)!;
  expect(Number(counts[1])).toBe(await page.locator('#search-results a').count());
  expect(Number(counts[2])).toBe(documents.length);
  expect(Number(counts[1])).toBeLessThan(Number(counts[2]));
  await query.fill('idempotency');
  await expect(page.locator('#search-results a').first()).toBeVisible();
  await expect(page.locator('#search-results')).toContainText(/idempotency/i);
  await query.fill('zzzz-no-matching-document');
  await expect(dialog.getByRole('status')).toContainText('No results');
  await expect(page.locator('#search-results a')).toHaveCount(0);
  await query.fill('backup');
  await expect(page.locator('#search-results a').first()).toContainText('Backups');
  await accessible(page);
  await noOverflow(page);
  await page.screenshot({ path: info.outputPath(`docs-search-${info.project.name}.png`) });
  await query.press('ArrowUp');
  await expect(page.locator('#search-results a').last()).toBeFocused();
  await query.focus();
  await query.press('ArrowDown');
  await expect(page.locator('#search-results a').first()).toBeFocused();
  await page.keyboard.press('End');
  await expect(page.locator('#search-results a').last()).toBeFocused();
  await page.keyboard.press('Tab');
  await expect(page.getByRole('button', { name: 'Close search' })).toBeFocused();
  await page.keyboard.press('Shift+Tab');
  await expect(page.locator('#search-results a').last()).toBeFocused();
  await page.keyboard.press('Home');
  await expect(page.locator('#search-results a').first()).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(dialog).not.toBeVisible();
  await expect(trigger).toBeFocused();
  await page.keyboard.press('Meta+k');
  await page.getByRole('button', { name: 'Close search' }).click();
  await expect(trigger).toBeFocused();
  await trigger.click();
  await query.fill('CLI reference');
  await expect(page.locator('#search-results a').first()).toContainText('CLI reference');
  await query.press('ArrowDown');
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(/\/docs\/cli\/$/);
});

test('copy code and Markdown use real clipboard and preserve focus; edit links target canonical sources', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.goto('docs/quickstart/');
  const code = await page.locator('article pre code').first().textContent();
  const copy = page.getByRole('button', { name: 'Copy code', exact: true }).first();
  await copy.focus(); await copy.press('Enter');
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe(code);
  await expect(copy).toBeFocused();
  const markdown = page.getByRole('button', { name: 'Copy as Markdown' });
  await markdown.click();
  const text = await page.evaluate(() => navigator.clipboard.readText());
  expect(text.startsWith('# Quickstart\n')).toBe(true);
  expect(text).toContain('## Patch with an expected revision');
  await expect(markdown).toBeFocused();
  await expect(page.locator('.docs-feedback')).toHaveText('Copied to clipboard.');
  await expect(page.getByRole('link', { name: 'Edit on GitHub' })).toHaveAttribute('href', 'https://github.com/matheus3301/kb/edit/main/docs/quickstart.md');
});

test('closing and reopening search restores an in-flight nonempty query', async ({ page }) => {
  const { promise: held, resolve: release } = Promise.withResolvers<void>();
  const { promise: started, resolve: requested } = Promise.withResolvers<void>();
  await page.route('**/docs/search.json', async route => { requested(); await held; await route.continue(); });
  await page.goto('docs/');
  const trigger = page.getByRole('button', { name: /Search docs/ });
  await trigger.click();
  await page.getByRole('searchbox').fill('backup');
  await started;
  await page.keyboard.press('Escape');
  await expect(trigger).toBeFocused();
  await trigger.click();
  await expect(page.getByRole('searchbox')).toHaveValue('backup');
  release();
  await expect(page.locator('#search-results a').first()).toContainText('Backups');
  await expect(page.getByRole('dialog').getByRole('status')).not.toContainText('Searching');
});

test('Markdown clipboard preserves literal angle brackets without executing markup', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.goto('docs/setup/');
  await page.getByRole('button', { name: 'Copy as Markdown' }).click();
  const copied = await page.evaluate(() => navigator.clipboard.readText());
  expect(copied).toContain('HOME/<PI_CONFIG_DIR>/profiles/<name>/agent');
  expect(copied).toContain('`<GEMINI_CLI_HOME>/.gemini`');
  await expect(page.locator('article pi_config_dir, article gemini_cli_home')).toHaveCount(0);
});

test('clipboard denial reports manual copy without false success', async ({ page }) => {
  await page.addInitScript(() => Object.defineProperty(navigator.clipboard, 'writeText', { value: () => Promise.reject(new DOMException('Denied', 'NotAllowedError')) }));
  await page.goto('docs/quickstart/');
  await page.getByRole('button', { name: 'Copy code', exact: true }).first().click();
  await expect(page.locator('.docs-feedback')).toHaveText('Clipboard unavailable. Select and copy the text manually.');
});

test('without JavaScript articles, source links, headings and mobile navigation remain usable', async ({ browser, baseURL, isMobile }, info) => {
  const context = await browser.newContext({ javaScriptEnabled: false, viewport: isMobile ? { width: 390, height: 844 } : { width: 1440, height: 1000 } });
  const page = await context.newPage();
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('response', response => { if (response.status() >= 400) errors.push(response.url()); });
  page.on('requestfailed', request => errors.push(request.url()));
  page.on('console', message => { if (message.type() === 'error') errors.push(message.text()); });
  await page.route('**/*', route => {
    const url = new URL(route.request().url());
    if (/^https?:$/.test(url.protocol) && url.origin !== new URL(baseURL!).origin) {
      errors.push(`External runtime request: ${url.href}`);
      return route.abort();
    }
    return route.continue();
  });
  await page.goto(`${baseURL}docs/`);
  await expect(page.getByRole('button', { name: /Search docs/ })).not.toBeVisible();
  await page.getByRole('navigation', { name: 'Documentation', exact: true }).getByRole('link', { name: 'Architecture', exact: true }).click();
  await expect(page.locator('article h1')).toHaveText('KB — Knowledge Base project design');
  await expect(page.locator('article')).toContainText('Mermaid source — diagram not rendered.');
  await expect(page.locator('article')).not.toContainText('generated:');
  await expect(page.locator('article pre').first()).toContainText('flowchart TD');
  if (isMobile) await page.locator('.docs-menu summary').click();
  await page.locator('article h2 a').first().click();
  await expect(page).toHaveURL(/#architecture$/);
  await noOverflow(page);
  await page.screenshot({ path: info.outputPath(`docs-no-js-${info.project.name}.png`) });
  expect(errors).toEqual([]);
  await context.close();
});
