export {};
type SearchEntry = { title: string; url: string; text: string };
const dialog = document.querySelector<HTMLDialogElement>('.docs-search')!;
const openButton = document.querySelector<HTMLButtonElement>('[data-search-open]')!;
const query = document.querySelector<HTMLInputElement>('#docs-query')!;
const results = document.querySelector<HTMLUListElement>('#search-results')!;
const status = document.querySelector<HTMLElement>('.search-status')!;
let returnFocus: HTMLElement | null = null;
let index: SearchEntry[] | undefined;
let pending: Promise<SearchEntry[]> | undefined;
let generation = 0;
openButton.hidden = false;

async function search() {
  const request = ++generation;
  const terms = query.value.toLocaleLowerCase().trim().split(/\s+/).filter(Boolean);
  results.replaceChildren();
  if (!terms.length) { status.textContent = 'Type to search. All terms must match.'; return; }
  status.textContent = 'Searching local documentation…';
  try {
    pending ??= fetch(dialog.dataset.index!).then(response => {
      if (!response.ok) throw new Error('Search index unavailable');
      return response.json() as Promise<SearchEntry[]>;
    });
    index ??= await pending;
    if (request !== generation) return;
    const matches = index.filter(entry => terms.every(term => `${entry.title} ${entry.text}`.toLocaleLowerCase().includes(term)))
      .sort((a, b) => score(b, terms) - score(a, terms) || a.title.localeCompare(b.title));
    const shown = matches.slice(0, 12);
    status.textContent = matches.length ? `Showing ${shown.length} of ${matches.length} matching documents.` : 'No results. Try fewer words or a command name.';
    for (const entry of shown) {
      const li = document.createElement('li');
      const a = document.createElement('a');
      a.href = entry.url;
      const title = document.createElement('strong'); title.textContent = entry.title;
      const excerpt = document.createElement('p');
      const position = Math.max(0, entry.text.toLocaleLowerCase().indexOf(terms[0]) - 65);
      excerpt.textContent = `${position ? '…' : ''}${entry.text.slice(position, position + 200)}…`;
      a.append(title, excerpt); li.append(a); results.append(li);
    }
  } catch {
    if (request !== generation) return;
    pending = undefined;
    status.textContent = 'Search is unavailable. Use the documentation menu; type again to retry.';
  }
}
function score(entry: SearchEntry, terms: string[]) {
  return terms.reduce((total, term) => total + (entry.title.toLocaleLowerCase().includes(term) ? 10 : 0), 0);
}
function openSearch() {
  if (dialog.open) return;
  returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : openButton;
  dialog.showModal(); query.focus();
  void search();
}
function closeSearch() { dialog.close(); }
openButton.addEventListener('click', openSearch);
document.querySelector('[data-search-close]')!.addEventListener('click', closeSearch);
dialog.addEventListener('close', () => { ++generation; returnFocus?.focus(); });
dialog.addEventListener('click', event => { if (event.target === dialog) { const rect = dialog.getBoundingClientRect(); if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) closeSearch(); } });
query.addEventListener('input', search);
document.addEventListener('keydown', event => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') { event.preventDefault(); openSearch(); }
});
dialog.addEventListener('keydown', event => {
  if (event.key === 'Escape') { event.preventDefault(); closeSearch(); return; }
  if (event.key === 'Tab') {
    const controls = [...dialog.querySelectorAll<HTMLElement>('button, input, a[href]')];
    const first = controls[0];
    const last = controls.at(-1);
    if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
    else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
    return;
  }
  const links = [...results.querySelectorAll<HTMLAnchorElement>('a')];
  const current = links.indexOf(document.activeElement as HTMLAnchorElement);
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault();
    if (links.length) {
      const next = current < 0
        ? (event.key === 'ArrowDown' ? 0 : links.length - 1)
        : (current + (event.key === 'ArrowDown' ? 1 : -1) + links.length) % links.length;
      links[next].focus();
    }
  } else if (event.key === 'Home' && current >= 0) { event.preventDefault(); links[0]?.focus(); }
  else if (event.key === 'End' && current >= 0) { event.preventDefault(); links.at(-1)?.focus(); }
  else if (event.key === 'Enter' && event.target === query && links.length) { event.preventDefault(); links[0].click(); }
});

const menu = document.querySelector<HTMLDetailsElement>('.docs-menu')!;
const mobile = matchMedia('(max-width: 760px)');
const adaptMenu = () => { menu.open = !mobile.matches; };
adaptMenu(); mobile.addEventListener('change', adaptMenu);
menu.addEventListener('keydown', event => { if (event.key === 'Escape' && mobile.matches) { menu.open = false; menu.querySelector('summary')?.focus(); } });

const feedback = document.querySelector<HTMLElement>('.docs-feedback')!;
async function copy(button: HTMLButtonElement, text: string) {
  if (button.getAttribute('aria-disabled') === 'true') return;
  button.setAttribute('aria-disabled', 'true');
  try { await navigator.clipboard.writeText(text); feedback.textContent = 'Copied to clipboard.'; }
  catch { feedback.textContent = 'Clipboard unavailable. Select and copy the text manually.'; }
  finally { button.removeAttribute('aria-disabled'); }
}
const markdownButton = document.querySelector<HTMLButtonElement>('[data-copy-markdown]')!;
markdownButton.hidden = false;
markdownButton.addEventListener('click', () => copy(markdownButton, JSON.parse(document.querySelector('#page-markdown')!.textContent!)));
for (const pre of document.querySelectorAll<HTMLPreElement>('.docs-prose pre')) {
  const code = pre.querySelector('code');
  if (!code) continue;
  const tools = document.createElement('div'); tools.className = 'docs-code-tools';
  const label = document.createElement('span');
  const language = [...code.classList].find(name => name.startsWith('language-'))?.slice(9);
  label.textContent = language === 'mermaid' ? 'Mermaid source · diagram not rendered' : 'Code example';
  const button = document.createElement('button'); button.textContent = 'Copy code';
  button.addEventListener('click', () => copy(button, code.textContent ?? ''));
  tools.append(label, button); pre.before(tools);
}
