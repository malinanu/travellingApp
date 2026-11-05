import type { CollectionConfig } from 'payload';

export const Features: CollectionConfig = {
  slug: 'features',
  admin: {
    useAsTitle: 'name',
    defaultColumns: ['name', 'category', 'icon'],
    group: 'Product & Pricing',
  },
  access: {
    read: () => true, // Publicly readable for website
  },
  fields: [
    {
      name: 'id',
      type: 'text',
      required: true,
      unique: true,
      label: 'Feature Key (e.g. feat_map_pin)',
    },
    {
      name: 'name',
      type: 'text',
      required: true,
      label: 'Feature Display Name',
    },
    {
      name: 'description',
      type: 'textarea',
      required: true,
      label: 'Feature Description',
    },
    {
      name: 'category',
      type: 'select',
      required: true,
      options: [
        { label: 'Discovery', value: 'discovery' },
        { label: 'Marketing & Urgency', value: 'marketing' },
        { label: 'Loyalty', value: 'loyalty' },
        { label: 'Operations', value: 'operations' },
        { label: 'Insights & Heatmaps', value: 'insights' },
        { label: 'Trust & Verification', value: 'trust' },
      ],
      defaultValue: 'discovery',
    },
    {
      name: 'icon',
      type: 'text',
      required: true,
      defaultValue: 'Sparkles',
      label: 'Lucide Icon Name',
    },
  ],
};
