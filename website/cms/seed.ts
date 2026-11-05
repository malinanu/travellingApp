/**
 * Payload CMS Seed & Synchronization Script
 * Seeds pages, pricing plans ($3.99, $11.99, $22.99), features, FAQs, testimonials, and stories.
 */
import { getPayload } from 'payload';
import config from './payload.config';
import seedData from './seed-data.json';

export async function seed() {
  const payload = await getPayload({ config });

  console.log('[Payload CMS Seed] Starting database seeding...');

  // 1. Seed Features
  const featureMap: Record<string, any> = {};
  for (const feat of seedData.features) {
    const existing = await payload.find({
      collection: 'features',
      where: { id: { equals: feat.id } },
      limit: 1,
    });
    if (existing.docs.length === 0) {
      const created = await payload.create({
        collection: 'features',
        data: feat as any,
      });
      featureMap[feat.id] = created.id;
      console.log(`[Seed] Created feature: ${feat.name}`);
    } else {
      featureMap[feat.id] = existing.docs[0].id;
    }
  }

  // 2. Seed Pricing Plans
  for (const plan of seedData.pricingPlans) {
    const existing = await payload.find({
      collection: 'pricing-plans',
      where: { slug: { equals: plan.slug } },
      limit: 1,
    });

    const planData = {
      ...plan,
      features: (plan.features || []).map((fId) => featureMap[fId] || fId),
      excludedFeatures: (plan.excludedFeatures || []).map((label) => ({ label })),
    };

    if (existing.docs.length === 0) {
      await payload.create({
        collection: 'pricing-plans',
        data: planData as any,
      });
      console.log(`[Seed] Created pricing plan: ${plan.name} ($${plan.monthlyPriceUsd}/mo)`);
    }
  }

  // 3. Seed FAQs
  for (const faq of seedData.faqs) {
    const existing = await payload.find({
      collection: 'faqs',
      where: { question: { equals: faq.question } },
      limit: 1,
    });
    if (existing.docs.length === 0) {
      await payload.create({
        collection: 'faqs',
        data: faq as any,
      });
      console.log(`[Seed] Created FAQ: ${faq.question}`);
    }
  }

  // 4. Seed Testimonials
  for (const t of seedData.testimonials) {
    const existing = await payload.find({
      collection: 'testimonials',
      where: { author: { equals: t.author } },
      limit: 1,
    });
    if (existing.docs.length === 0) {
      await payload.create({
        collection: 'testimonials',
        data: t as any,
      });
      console.log(`[Seed] Created testimonial from: ${t.author}`);
    }
  }

  // 5. Seed Pages
  for (const p of seedData.pages) {
    const existing = await payload.find({
      collection: 'pages',
      where: { slug: { equals: p.slug } },
      limit: 1,
    });
    if (existing.docs.length === 0) {
      await payload.create({
        collection: 'pages',
        data: p as any,
      });
      console.log(`[Seed] Created marketing page: ${p.slug}`);
    }
  }

  // 6. Seed Stories
  for (const s of seedData.stories) {
    const existing = await payload.find({
      collection: 'stories',
      where: { slug: { equals: s.slug } },
      limit: 1,
    });
    if (existing.docs.length === 0) {
      await payload.create({
        collection: 'stories',
        data: {
          ...s,
          tags: s.tags.map((tag) => ({ tag })),
        } as any,
      });
      console.log(`[Seed] Created story: ${s.title}`);
    }
  }

  console.log('[Payload CMS Seed] Finished seeding successfully.');
}

if (process.argv.includes('--run')) {
  seed()
    .then(() => process.exit(0))
    .catch((err) => {
      console.error('[Payload CMS Seed] Failed:', err);
      process.exit(1);
    });
}
