import path from 'path';
import { fileURLToPath } from 'url';
import { buildConfig } from 'payload';
import { postgresAdapter } from '@payloadcms/db-postgres';

const filename = fileURLToPath(import.meta.url);
const dirname = path.dirname(filename);
import { Users } from './collections/Users';
import { Pages } from './collections/Pages';
import { PricingPlans } from './collections/PricingPlans';
import { Features } from './collections/Features';
import { FAQs } from './collections/FAQs';
import { Testimonials } from './collections/Testimonials';
import { Stories } from './collections/Stories';

export default buildConfig({
  admin: {
    user: 'users',
    meta: {
      titleSuffix: '— Spotto CMS',
      icons: [
        {
          rel: 'icon',
          url: '/assets/icon.svg',
        },
      ],
    },
  },
  cors: [
    'http://localhost:3000',
    'http://127.0.0.1:3000',
    'http://localhost:5500',
    'http://127.0.0.1:5500',
    'http://localhost:8080',
    'http://127.0.0.1:8080',
    'https://spotto.dev',
    '*',
  ],
  csrf: [
    'http://localhost:3000',
    'http://127.0.0.1:3000',
    'http://localhost:5500',
    'http://127.0.0.1:5500',
    'http://localhost:8080',
  ],
  collections: [
    Users,
    Pages,
    PricingPlans,
    Features,
    FAQs,
    Testimonials,
    Stories,
  ],
  db: postgresAdapter({
    pool: {
      connectionString: process.env.DATABASE_URL || 'postgres://postgres:admin123@localhost:5432/spotto?sslmode=disable',
    },
  }),
  typescript: {
    outputFile: path.resolve(dirname, 'payload-types.ts'),
  },
});
