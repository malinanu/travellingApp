import type { CollectionConfig } from 'payload';

export const Testimonials: CollectionConfig = {
  slug: 'testimonials',
  admin: {
    useAsTitle: 'author',
    defaultColumns: ['author', 'venue', 'town', 'page', 'sortOrder'],
    group: 'Website Content',
  },
  access: {
    read: () => true, // Publicly readable for marketing site
  },
  fields: [
    {
      name: 'author',
      type: 'text',
      required: true,
      label: 'Author Name',
    },
    {
      name: 'quote',
      type: 'textarea',
      required: true,
      label: 'Quote Text',
    },
    {
      name: 'highlightQuote',
      type: 'text',
      label: 'Highlighted phrase within the quote',
    },
    {
      name: 'venue',
      type: 'text',
      required: true,
      label: 'Venue / Business / Description (e.g. Backpacker, Lyon or Owner, Nana\'s Kottu Shack)',
    },
    {
      name: 'town',
      type: 'text',
      label: 'Town / Region (e.g. Ella or Southern Province)',
    },
    {
      name: 'avatarInitials',
      type: 'text',
      label: 'Avatar Initials (e.g. LF, NP)',
    },
    {
      name: 'avatarUrl',
      type: 'text',
      label: 'Avatar Photo URL (optional)',
    },
    {
      name: 'page',
      type: 'select',
      options: [
        { label: 'Home Page', value: 'home' },
        { label: 'Businesses Page', value: 'businesses' },
        { label: 'Pricing Page', value: 'pricing' },
        { label: 'All Pages', value: 'all' },
      ],
      defaultValue: 'home',
      label: 'Display on Page',
    },
    {
      name: 'isPrimary',
      type: 'checkbox',
      defaultValue: false,
      label: 'Featured Main Quote (large typography)',
    },
    {
      name: 'rating',
      type: 'number',
      defaultValue: 5,
      label: 'Star Rating (1-5)',
    },
    {
      name: 'sortOrder',
      type: 'number',
      defaultValue: 1,
      label: 'Display Order',
    },
  ],
};
