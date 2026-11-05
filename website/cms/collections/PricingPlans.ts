import type { CollectionConfig } from 'payload';

export const PricingPlans: CollectionConfig = {
  slug: 'pricing-plans',
  admin: {
    useAsTitle: 'name',
    defaultColumns: ['name', 'slug', 'monthlyPriceUsd', 'yearlyPriceUsd', 'badge'],
    group: 'Product & Pricing',
  },
  access: {
    read: () => true, // Publicly readable for website
  },
  fields: [
    {
      name: 'name',
      type: 'text',
      required: true,
      label: 'Tier Name (e.g. Starter, Growth, Pro)',
    },
    {
      name: 'slug',
      type: 'text',
      required: true,
      unique: true,
      label: 'Tier Slug (starter, growth, pro)',
    },
    {
      name: 'tagline',
      type: 'text',
      required: true,
      label: 'Tagline / Value Proposition',
    },
    {
      type: 'row',
      fields: [
        {
          name: 'monthlyPriceUsd',
          type: 'number',
          required: true,
          label: 'Monthly Price ($ USD)',
          defaultValue: 3.99,
        },
        {
          name: 'yearlyPriceUsd',
          type: 'number',
          required: true,
          label: 'Yearly Price ($ USD)',
          defaultValue: 39.9,
        },
      ],
    },
    {
      type: 'row',
      fields: [
        {
          name: 'badge',
          type: 'text',
          label: 'Badge Label (e.g. Most Popular, Best Value)',
        },
        {
          name: 'highlighted',
          type: 'checkbox',
          label: 'Highlight Card (Featured Styling)',
          defaultValue: false,
        },
      ],
    },
    {
      type: 'row',
      fields: [
        {
          name: 'activeOffers',
          type: 'number',
          label: 'Active Offers Limit (-1 for unlimited)',
          defaultValue: 1,
        },
        {
          name: 'flashSalesPerMonth',
          type: 'number',
          label: 'Flash Sales Limit (-1 for unlimited)',
          defaultValue: 0,
        },
        {
          name: 'teamSeats',
          type: 'number',
          label: 'Team Seats Limit (-1 for unlimited)',
          defaultValue: 1,
        },
      ],
    },
    {
      name: 'features',
      type: 'relationship',
      relationTo: 'features',
      hasMany: true,
      label: 'Allocated System Features',
    },
    {
      name: 'excludedFeatures',
      type: 'array',
      label: 'Excluded Feature Labels (Dimmed with cross)',
      fields: [
        {
          name: 'label',
          type: 'text',
          required: true,
        },
      ],
    },
    {
      name: 'ctaText',
      type: 'text',
      defaultValue: 'Get Started',
      label: 'Call to Action Button Text',
    },
    {
      name: 'ctaLink',
      type: 'text',
      defaultValue: 'businesses.html#get-started',
      label: 'Call to Action Destination Link',
    },
    {
      name: 'sortOrder',
      type: 'number',
      defaultValue: 1,
      label: 'Sort Order',
    },
  ],
  hooks: {
    afterChange: [
      async ({ doc, operation }) => {
        // Optional webhook / sync hook to notify Spotto Go API server on CMS edit
        const spottoApiUrl = process.env.SPOTTO_API_URL || 'http://localhost:8080';
        try {
          console.log(`[Payload CMS] Pricing plan ${doc.name} updated via CMS (${operation}).`);
        } catch (err) {
          console.error('[Payload CMS] Sync error:', err);
        }
      },
    ],
  },
};
