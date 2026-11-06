import AxeBuilder from '@axe-core/playwright';
import { expect, test as base, type Locator, type Page } from '@playwright/test';


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

test('copy controls write the complete visible commands and agent prompt to the real clipboard', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await openLanding(page);
  for (const audience of ['For You', 'For Your Agent']) {
    await page.getByRole('tab', { name: audience, exact: true }).click();
    const panel = page.getByRole('tabpanel', { name: audience, exact: true });
    for (const block of await panel.locator('.copy-block').all()) {
      const text = await block.locator('pre').innerText();
      const copy = block.getByRole('button');
      await copy.press('Enter');
      await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe(text);
      await expect(block.getByRole('status')).toHaveText('Copied to clipboard.');
      await expectVisibleFocus(copy);
    }
  }
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
  const copy = page.getByRole('button', { name: 'Copy install and set up kb', exact: true });
  await copy.press('Enter');
  await expect(page.locator('html')).toHaveAttribute('data-copy-attempts', '1');
  await expectVisibleFocus(copy);
  await page.keyboard.press('Enter');
  await expect(page.locator('html')).toHaveAttribute('data-copy-attempts', '1');
  await page.keyboard.press('Tab');
  const commands = page.getByLabel('Install and set up KB', { exact: true });
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
    ['For You', 'Copy install and set up kb'],
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

test('foundations expose primary sources between benefits and documentation', async ({ page }) => {
  await openLanding(page);
  const foundations = page.getByRole('region').filter({ has: page.locator('#foundations-title') });
  await expect(foundations.getByRole('heading', { level: 2 })).toBeVisible();
  const okf = foundations.getByRole('article').filter({
    has: page.getByRole('heading', { name: /Open Knowledge Format/ }),
  });
  const wiki = foundations.getByRole('article').filter({
    has: page.getByRole('heading', { name: /LLM Wiki/ }),
  });
  await expect(okf).toBeVisible();
  await expect(wiki).toBeVisible();
  for (const href of [
    'https://github.com/GoogleCloudPlatform/open-knowledge-format',
    'https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md',
  ]) {
    await expect(okf.locator(`a[href="${href}"]`)).toBeVisible();
  }
  await expect(wiki.locator('a[href="https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f"]'))
    .toBeVisible();
  const placement = await foundations.evaluate(section => ({
    afterBenefits: section.previousElementSibling?.getAttribute('aria-labelledby') === 'benefits-title',
    beforeDocs: section.nextElementSibling?.id === 'docs',
  }));
  expect(placement).toEqual({ afterBenefits: true, beforeDocs: true });
  await foundations.scrollIntoViewIfNeeded();
  await expectNoPageOverflow(page);
});

test('brand images decode as self-hosted WebP and compact PNG favicons preserve explicit sizes', async ({ page }) => {
  await openLanding(page);
  for (const [selector, width, height] of [
    ['header .brand img', 40, 40],
    ['footer .brand img', 30, 30],
  ] as const) {
    const logo = page.locator(selector);
    await expect(logo).toHaveAttribute('alt', '');
    await expect(logo).toHaveAttribute('width', String(width));
    await expect(logo).toHaveAttribute('height', String(height));
    await expect(logo.locator('..')).toContainText('Knowledge Base');
    await logo.scrollIntoViewIfNeeded();
    // A newly visible lazy image must start loading before decode() can succeed.
    await expect.poll(() => logo.evaluate(element => {
      const image = element as HTMLImageElement;
      return image.complete && image.naturalWidth > 0;
    })).toBe(true);
    const image = await logo.evaluate(async element => {
      const image = element as HTMLImageElement;
      await image.decode();
      const candidates = [image.src, ...image.srcset.split(',').filter(Boolean)
        .map(candidate => new URL(candidate.trim().split(/\s+/)[0], location.href).href)];
      const response = await fetch(image.currentSrc);
      const rect = image.getBoundingClientRect();
      return {
        decoded: image.complete && image.naturalWidth > 0 && image.naturalHeight > 0,
        localWebP: candidates.every(candidate => {
          const url = new URL(candidate);
          return url.origin === location.origin && url.pathname.endsWith('.webp');
        }),
        hasSrcset: image.srcset.length > 0,
        mime: response.headers.get('content-type')?.split(';')[0],
        bytes: (await response.arrayBuffer()).byteLength,
        width: rect.width,
        height: rect.height,
      };
    });
    expect(image.decoded).toBe(true);
    expect(image.localWebP).toBe(true);
    expect(image.hasSrcset).toBe(true);
    expect(image.mime).toBe('image/webp');
    expect(image.bytes).toBeLessThan(32 * 1024);
    const mobileHeader = selector.startsWith('header') && (page.viewportSize()?.width ?? 1440) <= 760;
    expect(image.width).toBeCloseTo(mobileHeader ? 30 : width, 1);
    expect(image.height).toBeCloseTo(mobileHeader ? 30 : height, 1);
  }
  const icons = page.locator('link[rel="icon"]');
  await expect(icons).toHaveCount(2);
  for (const size of [32, 64]) {
    const icon = page.locator(`link[rel="icon"][sizes="${size}x${size}"]`);
    await expect(icon).toHaveAttribute('type', 'image/png');
    const metadata = await icon.evaluate(async element => {
      const url = new URL((element as HTMLLinkElement).href);
      const response = await fetch(url);
      const bytes = await response.arrayBuffer();
      const image = new Image();
      image.src = url.href;
      await image.decode();
      return {
        local: url.origin === location.origin,
        mime: response.headers.get('content-type')?.split(';')[0],
        bytes: bytes.byteLength,
        width: image.naturalWidth,
        height: image.naturalHeight,
      };
    });
    expect(metadata.local).toBe(true);
    expect(metadata.mime).toBe('image/png');
    expect(metadata.bytes).toBeGreaterThan(0);
    expect(metadata.bytes).toBeLessThan(32 * 1024);
    expect(metadata.width).toBe(size);
    expect(metadata.height).toBe(size);
  }
  const largeOriginal = await page.evaluate(() => {
    const icons = new Set([...document.querySelectorAll<HTMLLinkElement>('link[rel="icon"]')]
      .map(icon => icon.href));
    return performance.getEntriesByType('resource')
      .map(entry => new URL(entry.name))
      .filter(url => /\.png$/i.test(url.pathname) && !icons.has(url.href))
      .map(url => url.pathname);
  });
  expect(largeOriginal, 'The original logo PNG must not be loaded by the page').toEqual([]);
  await expectNoPageOverflow(page);
});

test('both setup views fit, pass serious/critical axe checks, and produce public screenshots', async ({ page }, testInfo) => {
  await openLanding(page);
  // Include the below-fold brand image in the actual page evidence.
  const footerLogo = page.locator('footer .brand img');
  await footerLogo.scrollIntoViewIfNeeded();
  await expect.poll(() => footerLogo.evaluate(element => {
    const image = element as HTMLImageElement;
    return image.complete && image.naturalWidth > 0;
  })).toBe(true);
  await page.evaluate(() => window.scrollTo(0, 0));
  await expectNoPageOverflow(page);
  await expectAccessible(page);
  await page.getByRole('tab', { name: 'For You', exact: true }).focus();
  await page.screenshot({ path: testInfo.outputPath(`${testInfo.project.name}.png`), fullPage: true });
  await page.locator('#get-started').screenshot({ path: testInfo.outputPath(`${testInfo.project.name}-setup.png`) });
  await page.getByRole('tab', { name: 'For Your Agent', exact: true }).click();
  await expectNoPageOverflow(page);
  await expectAccessible(page);
  await page.getByRole('tab', { name: 'For Your Agent', exact: true }).focus();
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: testInfo.outputPath(`${testInfo.project.name}-agent.png`), fullPage: true });
  await page.locator('#get-started').screenshot({ path: testInfo.outputPath(`${testInfo.project.name}-agent-setup.png`) });
});

test('reduced motion shows a complete, keyboard-scrollable terminal transcript', async ({ page }, testInfo) => {
  await page.addInitScript(() => { Math.random = () => 0; });
  await openLanding(page);
  const demo = page.getByRole('region', { name: 'KB terminal' });
  const transcript = demo.getByRole('region', { name: 'Session transcript' });
  const next = demo.getByRole('button', { name: 'Next example' });
  await expect(transcript).toContainText('worker’s memory budget');
  await expect(demo.getByRole('button', { name: 'Pause', exact: true })).toHaveCount(0);

  await page.reload();
  await expect(transcript).toContainText('HTTP 401');
  await next.click();
  await expect(next).toBeFocused();
  await expect(transcript.getByText('kb patch --input patch.json --dry-run', { exact: true })).toBeVisible();
  await expect(transcript.getByText('Keep the previous signing key active for 24 hours.', { exact: false }).first()).toBeVisible();
  await expect(transcript.getByText('I updated the rotation runbook.', { exact: false })).toBeVisible();
  const citation = transcript.getByText('03_Resources/webhook-rotation.md:11', { exact: true });
  await transcript.press('End');
  await expect(citation).toBeInViewport();
  await expectNoPageOverflow(page);
  await expectAccessible(page);
  await demo.screenshot({ path: testInfo.outputPath(`${testInfo.project.name}-terminal-write.png`) });
});

test('the animated terminal accumulates calls and supports pause, replay, and automatic advance', async ({ page }, testInfo) => {
  await page.emulateMedia({ reducedMotion: 'no-preference' });
  await page.addInitScript(() => { Math.random = () => 0; });
  await page.clock.install();
  await openLanding(page);
  const demo = page.getByRole('region', { name: 'KB terminal' });
  const transcript = demo.getByRole('region', { name: 'Session transcript' });
  const search = transcript.getByText('kb search "export batches" --limit 3', { exact: true });
  const read = transcript.getByText('kb read 01_Projects/exports/batching.md', { exact: true });
  const isPlaying = () => demo.evaluate(element => element.hasAttribute('data-playing'));
  await demo.scrollIntoViewIfNeeded();
  await expect.poll(isPlaying).toBe(true);
  await expect(transcript.getByText('Why did we cap export batches at 500 rows?')).toBeVisible();
  await expect(search).toBeHidden();

  await page.clock.runFor(2400);
  await expect(search).toBeVisible();
  await demo.getByRole('button', { name: 'Pause', exact: true }).click();
  await page.clock.fastForward(30_000);
  await expect(read).toBeHidden();

  await demo.getByRole('button', { name: 'Play', exact: true }).click();
  await page.clock.runFor(3300);
  await expect(search).toBeVisible();
  await expect(read).toBeVisible();
  await demo.getByRole('button', { name: 'Replay', exact: true }).click();
  await expect(read).toBeHidden();
  await expect(search).toBeHidden();
  await page.clock.runFor(10_000);
  await expect(search).toBeVisible();
  await expect(read).toBeVisible();
  await expect(transcript).toContainText('The 500-row cap keeps retries within the worker’s memory budget.');
  await expect(transcript.getByText('01_Projects/exports/batching.md:8').last()).toBeInViewport();
  await transcript.press('Home');
  await expect(demo.getByRole('button', { name: 'Play', exact: true })).toBeVisible();
  await expect(transcript.getByText('Why did we cap export batches at 500 rows?')).toBeInViewport();
  await transcript.press('End');
  await expect(transcript.getByText('01_Projects/exports/batching.md:8').last()).toBeInViewport();
  await demo.screenshot({ path: testInfo.outputPath(`${testInfo.project.name}-terminal-reply.png`) });
  await demo.getByRole('button', { name: 'Play', exact: true }).click();
  await page.clock.runFor(9000);
  await expect(transcript.getByText('The nightly import stalled again. What should I check first?')).toBeVisible();

  await page.locator('footer.site-footer').scrollIntoViewIfNeeded();
  await expect.poll(isPlaying).toBe(false);
  await page.clock.fastForward(30_000);
  await expect(transcript.getByText('kb context "nightly import stalled" --max-tokens 600')).toBeHidden();
});

test.describe('without JavaScript', () => {
  test.use({ javaScriptEnabled: false });

  test('the initial prompt, commands, answer, and source remain readable', async ({ page }) => {
    await page.goto('./');
    const demo = page.getByRole('region', { name: 'KB terminal' });
    const transcript = demo.getByRole('region', { name: 'Session transcript' });
    await expect(transcript.getByText('Why did we cap export batches at 500 rows?')).toBeVisible();
    await expect(transcript.getByText('kb read 01_Projects/exports/batching.md')).toBeVisible();
    await expect(transcript.getByText('The 500-row cap keeps retries within the worker’s memory budget.', { exact: false })).toBeVisible();
    await expect(transcript.getByText('01_Projects/exports/batching.md:8').last()).toBeVisible();
    await expect(demo.getByRole('button')).toHaveCount(0);
    await expectNoPageOverflow(page);
  });
});
