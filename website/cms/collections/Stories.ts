import type { CollectionConfig } from 'payload';

export const Stories: CollectionConfig = {
  slug: 'stories',
  admin: {
    useAsTitle: 'title',
    defaultColumns: ['title', 'slug', 'category', 'status', 'publishedAt'],
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
      label: 'Story Title',
    },
    {
      name: 'slug',
      type: 'text',
      required: true,
      unique: true,
      label: 'URL Slug',
    },
    {
      name: 'summary',
      type: 'textarea',
      label: 'Story Summary / Excerpt',
    },
    {
      name: 'body',
      type: 'textarea',
      required: true,
      label: 'Article Body (Markdown)',
    },
    {
      name: 'coverUrl',
      type: 'text',
      label: 'Cover Image URL',
    },
    {
      name: 'category',
      type: 'text',
      defaultValue: 'Story',
      label: 'Category Tag (Story, Guide, Food, Surf)',
    },
    {
      name: 'authorName',
      type: 'text',
      label: 'Author Name',
    },
    {
      name: 'regionName',
      type: 'text',
      label: 'Region Name (e.g. Arugam Bay, Ella, Galle)',
    },
    {
      name: 'tags',
      type: 'array',
      label: 'Tags',
      fields: [
        {
          name: 'tag',
          type: 'text',
          required: true,
        },
      ],
    },
    {
      name: 'featured',
      type: 'checkbox',
      defaultValue: false,
      label: 'Feature as Main Header Story',
    },
    {
      name: 'status',
      type: 'select',
      options: [
        { label: 'Published', value: 'published' },
        { label: 'Draft', value: 'draft' },
      ],
      defaultValue: 'published',
      required: true,
    },
    {
      name: 'publishedAt',
      type: 'date',
      label: 'Publication Date',
    },
  ],
};
