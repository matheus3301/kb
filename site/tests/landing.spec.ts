import AxeBuilder from '@axe-core/playwright';
import { expect, test as base, type Locator, type Page } from '@playwright/test';

const installCommands = 'brew install matheus3301/tap/kb\nnpx skills add matheus3301/kb --skill knowledge-base';
const baseCommands = 'export KB_ROOT="$PWD/example-kb"\nkb init --para\nkb status';

// A site visit must not need third-party assets, APIs, or analytics.
const test = base.extend({
  page: async ({ page }, use) => {
    const errors: string[] = [];
    const externalRequests: string[] = [];
    page.on('pageerror', error => errors.push(error.message));
    page.on('console', message => {
      if (message.type() === 'error') errors.push(message.text());
    });
    page.on('response', response => {
      if (response.status() >= 400) errors.push(`${response.status()} ${response.url()}`);
    });
    await page.route('**/*', route => {
      const url = new URL(route.request().url());
      if (url.protocol === 'http:' || url.protocol === 'https:') {
        if (url.origin !== 'http://127.0.0.1:4321') {
          externalRequests.push(url.href);
          return route.abort();
        }
      }
      return route.continue();
    });
    await use(page);
    expect(externalRequests, 'Runtime resources must be self-hosted').toEqual([]);
    expect(errors, 'The production page must not emit browser or HTTP errors').toEqual([]);
  },
});

async function openLanding(page: Page) {
  await page.goto('./');
  await expect(page.getByRole('heading', { level: 1 })).toContainText('Your Knowledge Base.');
  await expect(page.getByRole('tab', { name: 'For You', exact: true })).toHaveAttribute('aria-selected', 'true');
  await page.evaluate(() => document.fonts.ready);
}

async function expectVisibleFocus(control: Locator) {
  await expect(control).toBeFocused();
  const focus = await control.evaluate(element => {
    const style = getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return {
      focusVisible: element.matches(':focus-visible'),
      outline: style.outlineStyle !== 'none' && parseFloat(style.outlineWidth) >= 2
        && style.outlineColor !== 'rgba(0, 0, 0, 0)',
      inViewport: rect.top >= 0 && rect.left >= 0
        && rect.bottom <= innerHeight && rect.right <= innerWidth,
    };
  });
  expect(focus).toEqual({ focusVisible: true, outline: true, inViewport: true });
}

async function expectNoPageOverflow(page: Page) {
  const overflow = await page.evaluate(() => Math.max(
    document.documentElement.scrollWidth, document.body.scrollWidth,
  ) - document.documentElement.clientWidth);
  expect(overflow, 'Content must fit the viewport without horizontal page scrolling').toBeLessThanOrEqual(1);
}

async function expectAccessible(page: Page) {
  const results = await new AxeBuilder({ page }).analyze();
  const failures = results.violations.filter(({ impact }) => impact === 'serious' || impact === 'critical');
  expect(failures, JSON.stringify(failures, null, 2)).toEqual([]);
}

test('install tabs support pointer selection, wraparound, Home, End, and visible keyboard focus', async ({ page }) => {
  await openLanding(page);
  const you = page.getByRole('tab', { name: 'For You', exact: true });
  const agent = page.getByRole('tab', { name: 'For Your Agent', exact: true });
  await expect(page.getByRole('tabpanel', { name: 'For You', exact: true })).toBeVisible();
  await agent.click();
  await expect(agent).toHaveAttribute('aria-selected', 'true');
  await expect(you).toHaveAttribute('tabindex', '-1');
  await expect(page.getByRole('tabpanel', { name: 'For Your Agent', exact: true })).toBeVisible();
  await expect(page.getByRole('tabpanel')).toHaveCount(1);
  await agent.press('Home');
  await expect(you).toHaveAttribute('aria-selected', 'true');
  await expectVisibleFocus(you);
  await you.press('ArrowLeft');
  await expect(agent).toHaveAttribute('aria-selected', 'true');
  await expectVisibleFocus(agent);
  await agent.press('ArrowRight');
  await expect(you).toHaveAttribute('aria-selected', 'true');
  await you.press('End');
  await expect(agent).toHaveAttribute('aria-selected', 'true');
  await expectVisibleFocus(agent);
  await agent.press('ArrowLeft');
  await expect(you).toHaveAttribute('aria-selected', 'true');
  await you.press('ArrowRight');
  await expect(agent).toHaveAttribute('aria-selected', 'true');
  await agent.press('Tab');
  await expect(page.getByRole('tabpanel', { name: 'For Your Agent', exact: true })).toBeFocused();
  await page.keyboard.press('Tab');
  await expectVisibleFocus(page.getByRole('button', { name: 'Copy setup prompt', exact: true }));
});

test('copy controls write real install commands, base setup, and the public agent prompt', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await openLanding(page);
  const you = page.getByRole('tabpanel', { name: 'For You', exact: true });
  const installCopy = you.getByRole('button', { name: 'Copy install cli + skill', exact: true });
  await installCopy.press('Enter');
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe(installCommands);
  await expect(you.getByRole('status').filter({ hasText: 'Copied to clipboard.' })).toHaveCount(1);
  await expectVisibleFocus(installCopy);
  await you.getByRole('button', { name: 'Copy create a new example base', exact: true }).click();
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe(baseCommands);
  await page.getByRole('tab', { name: 'For Your Agent', exact: true }).click();
  const agent = page.getByRole('tabpanel', { name: 'For Your Agent', exact: true });
  await agent.getByRole('button', { name: 'Copy setup prompt', exact: true }).click();
  await expect(agent.getByRole('status')).toHaveText('Copied to clipboard.');
  const prompt = await page.evaluate(() => navigator.clipboard.readText());
  expect(prompt).toContain('https://raw.githubusercontent.com/matheus3301/kb/main/docs/agent-setup.md');
  expect(prompt).toContain('https://raw.githubusercontent.com/matheus3301/kb/main/llms.txt');
  expect(prompt).toContain('brew install matheus3301/tap/kb');
  expect(prompt).toContain('npx skills add matheus3301/kb --skill knowledge-base');
  expect(prompt).toMatch(/preserve existing knowledge bases/i);
  expect(prompt).toMatch(/base and workdir/i);
  expect(prompt).toMatch(/skill installation alone does not activate/i);
  expect(prompt).toMatch(/verify kb version, the installed skill/i);
  expect(prompt).toMatch(/actual search\/read/i);
});

test('pending copy preserves keyboard focus, guards repeat activation, and respects later focus movement', async ({ page }) => {
  await page.addInitScript(() => {
    let attempts = 0;
    let finish: (() => void) | undefined;
    Object.defineProperty(navigator.clipboard, 'writeText', {
      value: () => {
        document.documentElement.dataset.copyAttempts = String(++attempts);
        return new Promise<void>(resolve => { finish = resolve; });
      },
    });
    window.addEventListener('kb-test-finish-copy', () => finish?.());
  });
  await openLanding(page);
  const copy = page.getByRole('button', { name: 'Copy install cli + skill', exact: true });
  await copy.press('Enter');
  await expect(page.locator('html')).toHaveAttribute('data-copy-attempts', '1');
  await expectVisibleFocus(copy);
  await page.keyboard.press('Enter');
  await expect(page.locator('html')).toHaveAttribute('data-copy-attempts', '1');
  await page.keyboard.press('Tab');
  const commands = page.getByLabel('Install CLI + skill', { exact: true });
  await expect(commands).toBeFocused();
  await page.evaluate(() => { window.dispatchEvent(new Event('kb-test-finish-copy')); });
  await expect(page.getByRole('status').filter({ hasText: 'Copied to clipboard.' })).toHaveCount(1);
  await expect(commands).toBeFocused();
});

test('denied clipboard access provides live manual-copy feedback without false success', async ({ page }) => {
  // Deterministically exercise the browser rejection, independent of host permissions.
  await page.addInitScript(() => {
    Object.defineProperty(navigator.clipboard, 'writeText', {
      value: async () => { throw new DOMException('Clipboard denied', 'NotAllowedError'); },
    });
  });
  await openLanding(page);
  for (const [tab, button] of [
    ['For You', 'Copy install cli + skill'],
    ['For Your Agent', 'Copy setup prompt'],
  ]) {
    await page.getByRole('tab', { name: tab, exact: true }).click();
    const panel = page.getByRole('tabpanel', { name: tab, exact: true });
    const copy = panel.getByRole('button', { name: button, exact: true });
    await copy.press('Enter');
    await expect(panel.getByRole('status').filter({ hasText: 'Clipboard unavailable.' }))
      .toHaveText('Clipboard unavailable. Select and copy the text manually.');
    await expect(panel.getByRole('status').filter({ hasText: 'Copied to clipboard.' })).toHaveCount(0);
    await expect(copy).toBeEnabled();
    await expectVisibleFocus(copy);
  }
});

test('header navigation and same-document anchors resolve at the deployed subpath', async ({ page }) => {
  await openLanding(page);
  const navigation = page.getByRole('navigation').first();
  await navigation.getByRole('link', { name: 'Docs', exact: true }).click();
  await expect(page).toHaveURL('http://127.0.0.1:4321/kb/#docs');
  await expect(page.locator('#docs')).toBeInViewport();
  await expect(navigation.getByRole('link', { name: 'GitHub', exact: true }))
    .toHaveAttribute('href', 'https://github.com/matheus3301/kb');
  const broken = await page.locator('a[href^="#"]').evaluateAll(links => links.flatMap(link => {
    const fragment = link.getAttribute('href')?.slice(1);
    return fragment && !document.getElementById(decodeURIComponent(fragment)) ? [fragment] : [];
  }));
  expect(broken, 'Only local page fragments are checked; remote URLs are not fetched').toEqual([]);
});

test('both setup views fit, pass serious/critical axe checks, and produce public screenshots', async ({ page }, testInfo) => {
  await openLanding(page);
  await expectNoPageOverflow(page);
  await expectAccessible(page);
  await page.screenshot({ path: testInfo.outputPath(`${testInfo.project.name}.png`), fullPage: true });
  await page.getByRole('tab', { name: 'For Your Agent', exact: true }).click();
  await expectNoPageOverflow(page);
  await expectAccessible(page);
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: testInfo.outputPath(`${testInfo.project.name}-agent.png`), fullPage: true });
});
