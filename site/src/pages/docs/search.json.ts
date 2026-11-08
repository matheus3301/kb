import { getCollection } from 'astro:content';
import { documents, docsPath } from '../../docs/catalog.mjs';

export async function GET() {
  const entries = await getCollection('docs');
  const index = entries.map(entry => ({
    title: documents.find(doc => doc.id === entry.id)!.title,
    url: docsPath(entry.id, import.meta.env.BASE_URL),
    text: (entry.body ?? '').replace(/```[^\n]*/g, '').replace(/\[([^\]]+)\]\([^)]+\)/g, '$1').replace(/[#*`>|]/g, '').replace(/\s+/g, ' ').trim(),
  }));
  return new Response(JSON.stringify(index), { headers: { 'Content-Type': 'application/json' } });
}
