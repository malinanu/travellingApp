import type { CollectionConfig } from 'payload';

export const FAQs: CollectionConfig = {
  slug: 'faqs',
  admin: {
    useAsTitle: 'question',
    defaultColumns: ['question', 'category', 'sortOrder'],
    group: 'Website Content',
  },
  access: {
    read: () => true,
  },
  fields: [
    {
      name: 'question',
      type: 'text',
      required: true,
      label: 'Question',
    },
    {
      name: 'answer',
      type: 'textarea',
      required: true,
      label: 'Answer',
    },
    {
      name: 'category',
      type: 'select',
      options: [
        { label: 'Pricing & Plans', value: 'pricing' },
        { label: 'For Businesses', value: 'businesses' },
        { label: 'Travellers & App', value: 'travellers' },
      ],
      defaultValue: 'pricing',
    },
    {
      name: 'sortOrder',
      type: 'number',
      defaultValue: 1,
    },
  ],
};
