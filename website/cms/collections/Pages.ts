import type { CollectionConfig } from 'payload';

export const Pages: CollectionConfig = {
  slug: 'pages',
  admin: {
    useAsTitle: 'title',
    defaultColumns: ['title', 'slug', 'updatedAt'],
    group: 'Website Content',
  },
  access: {
    read: () => true, // Publicly readable for website
  },
  fields: [
    {
      name: 'title',
      type: 'text',
      required: true,
      label: 'Page Title',
    },
    {
      name: 'slug',
      type: 'text',
      required: true,
      unique: true,
      label: 'Page URL Slug (home, features, businesses, pricing, about, contact)',
    },
    {
      name: 'metaDescription',
      type: 'textarea',
      label: 'SEO Meta Description',
    },
    {
      name: 'hero',
      type: 'group',
      label: 'Hero Section',
      fields: [
        {
          name: 'eyebrow',
          type: 'text',
          label: 'Eyebrow Tag (e.g. "Made in Sri Lanka" or "For businesses")',
        },
        {
          name: 'headline',
          type: 'text',
          required: true,
          label: 'H1 Headline',
        },
        {
          name: 'highlightText',
          type: 'text',
          label: 'Word(s) to highlight with golden hand-drawn squiggle',
        },
        {
          name: 'lead',
          type: 'textarea',
          label: 'Lead Paragraph',
        },
        {
          name: 'primaryCtaText',
          type: 'text',
          label: 'Primary CTA Text',
        },
        {
          name: 'primaryCtaLink',
          type: 'text',
          label: 'Primary CTA Link',
        },
        {
          name: 'secondaryCtaText',
          type: 'text',
          label: 'Secondary CTA Text',
        },
        {
          name: 'secondaryCtaLink',
          type: 'text',
          label: 'Secondary CTA Link',
        },
      ],
    },
    {
      name: 'sections',
      type: 'blocks',
      label: 'Page Content Blocks',
      blocks: [
        {
          slug: 'statsRow',
          labels: { singular: 'Stats Row', plural: 'Stats Rows' },
          fields: [
            {
              name: 'stats',
              type: 'array',
              fields: [
                { name: 'value', type: 'text', required: true, label: 'Stat Value (e.g. 6,800+)' },
                { name: 'label', type: 'text', required: true, label: 'Stat Label (e.g. Active travellers)' },
              ],
            },
          ],
        },
        {
          slug: 'splitFeature',
          labels: { singular: 'Split Content Row', plural: 'Split Content Rows' },
          fields: [
            { name: 'eyebrow', type: 'text', label: 'Eyebrow' },
            { name: 'title', type: 'text', required: true, label: 'Section Title' },
            { name: 'body', type: 'textarea', required: true, label: 'Body Text' },
            {
              name: 'bullets',
              type: 'array',
              fields: [{ name: 'text', type: 'text', required: true }],
            },
            { name: 'imageTag', type: 'text', label: 'Photo Caption / Place Tag' },
          ],
        },
        {
          slug: 'testimonials',
          labels: { singular: 'Testimonials Group', plural: 'Testimonials Groups' },
          fields: [
            {
              name: 'items',
              type: 'array',
              fields: [
                { name: 'quote', type: 'textarea', required: true },
                { name: 'author', type: 'text', required: true },
                { name: 'venue', type: 'text', required: true },
                { name: 'town', type: 'text', required: true },
              ],
            },
          ],
        },
        {
          slug: 'ctaBand',
          labels: { singular: 'CTA Banner Band', plural: 'CTA Banner Bands' },
          fields: [
            { name: 'headline', type: 'text', required: true },
            { name: 'subcopy', type: 'textarea' },
            { name: 'buttonText', type: 'text', required: true },
            { name: 'buttonLink', type: 'text', required: true },
          ],
        },
      ],
    },
  ],
};
