import { defineCollection } from 'astro:content';
import { glob } from 'astro/loaders';
import { documents } from './docs/catalog.mjs';

export const collections = {
  docs: defineCollection({
    loader: glob({
      base: '../docs',
      pattern: documents.map(({ id }) => `${id}.md`),
      generateId: ({ entry }) => entry.replace(/\.md$/, ''),
    }),
  }),
};
