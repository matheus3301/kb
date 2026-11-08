import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { documents, docsPath, repository } from './catalog.mjs';

const root = fileURLToPath(new URL('../../../', import.meta.url));
const published = new Set(documents.map(({ id }) => `docs/${id}.md`));

export function resolveDocLink(href, source, base) {
  if (/[\u0000-\u0020\\]/.test(href)) return null;
  let decoded;
  const match = href.match(/^([^?#]*)(\?[^#]*)?(#.*)?$/);
  if (!match) return null;
  try { decoded = decodeURIComponent(match[1]); } catch { return null; }
  if (/[\u0000-\u0020\\]/.test(decoded) || decoded.startsWith('/') || decoded.includes('%')) return null;
  if (/^[a-z][a-z\d+.-]*:/i.test(decoded)) return /^(https?:|mailto:)/i.test(href) ? href : null;
  if (decoded !== match[1] && decoded.split('/').includes('..')) return null;
  const suffix = `${match[2] ?? ''}${match[3] ?? ''}`;
  if (!decoded) return suffix || null;
  const target = path.posix.normalize(path.posix.join(path.posix.dirname(source), decoded));
  if (target.startsWith('../') || path.posix.isAbsolute(target)) return null;
  if (published.has(target)) return `${docsPath(target.slice(5, -3), base)}${suffix}`;
  return `${repository}/blob/main/${target.split('/').map(encodeURIComponent).join('/')}${suffix}`;
}

export const safeDocs = {
  name: 'kb-safe-docs',
  html(node, ctx) { ctx.removeNode(node); },
  image(node, ctx) { ctx.replaceNode(node, { type: 'text', value: node.alt ?? '' }); },
  imageReference(node, ctx) { ctx.replaceNode(node, { type: 'text', value: node.alt ?? '' }); },
  code(node, ctx) {
    if (node.lang === 'mermaid') ctx.insertBefore(node, {
      type: 'paragraph', children: [{ type: 'text', value: 'Mermaid source — diagram not rendered.' }],
    });
  },
  link(node, ctx) { rewrite(node, ctx); },
  definition(node, ctx) { rewrite(node, ctx); },
};

function rewrite(node, ctx) {
  if (!ctx.fileURL) throw new Error('Documentation rendering requires a source path');
  const source = path.relative(root, fileURLToPath(ctx.fileURL)).split(path.sep).join('/');
  const url = resolveDocLink(node.url, source, process.env.KB_SITE_BASE ?? '/kb/');
  if (url === null && node.type === 'link') ctx.replaceNode(node, node.children);
  else ctx.setProperty(node, 'url', url ?? '');
}

export const readingSurface = {
  name: 'kb-reading-surface',
  element: [
    {
      filter: ['a'],
      visit(node, ctx) {
        if (!node.properties.href) ctx.replaceNode(node, node.children);
      },
    },
    {
      filter: ['h1', 'h2', 'h3', 'h4', 'h5', 'h6'],
      visit(node, ctx) {
        const id = node.properties.id;
        if (id) ctx.appendChild(node, {
          type: 'element', tagName: 'a',
          properties: { href: `#${id}`, className: ['heading-anchor'], ariaLabel: `Link to ${ctx.textContent(node)}` },
          children: [],
        });
      },
    },
    {
      filter: ['table'],
      visit(node, ctx) {
        ctx.wrapNode(node, { type: 'element', tagName: 'div', properties: { className: ['table-scroll'], tabIndex: 0, role: 'region', ariaLabel: 'Scrollable reference table' }, children: [] });
      },
    },
    {
      filter: ['pre'],
      visit(node, ctx) {
        ctx.setProperty(node, 'tabIndex', 0);
        ctx.setProperty(node, 'ariaLabel', 'Code example');
      },
    },
  ],
};
