/**
 * Spotto CMS <-> Command Center Sync Bridge
 * Bi-directionally synchronizes vendor packages and features between:
 * - Spotto Go API / Command Center (Postgres `platform_settings`)
 * - Payload CMS (`pricing-plans` and `features` collections)
 */

export interface SyncOptions {
  spottoApiUrl?: string;
  cmsApiUrl?: string;
  adminToken?: string;
}

export async function syncCommandCenterToCms(options: SyncOptions = {}) {
  const spottoApi = (options.spottoApiUrl || process.env.SPOTTO_API_URL || 'http://localhost:8080').replace(/\/$/, '');
  const cmsApi = (options.cmsApiUrl || process.env.PAYLOAD_CMS_URL || 'http://localhost:3002').replace(/\/$/, '');

  console.log(`[Sync Bridge] Fetching packages from Spotto API: ${spottoApi}/packages...`);
  const res = await fetch(`${spottoApi}/packages`, {
    headers: { Accept: 'application/json' },
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch packages from Spotto API: HTTP ${res.status}`);
  }

  const data = await res.json();
  const packages = data.packages || [];
  const featurePool = data.featurePool || [];

  console.log(`[Sync Bridge] Received ${packages.length} packages and ${featurePool.length} features.`);

  // 1. Sync Features to CMS
  for (const feat of featurePool) {
    try {
      const checkRes = await fetch(`${cmsApi}/api/features?where[id][equals]=${encodeURIComponent(feat.id)}`);
      if (checkRes.ok) {
        const checkJson = await checkRes.json();
        if (checkJson.totalDocs === 0) {
          await fetch(`${cmsApi}/api/features`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(feat),
          });
          console.log(`[Sync Bridge] Created CMS feature: ${feat.name}`);
        }
      }
    } catch (e) {
      console.warn(`[Sync Bridge] Feature sync warning for ${feat.id}:`, e);
    }
  }

  // 2. Sync Packages to CMS
  for (const pkg of packages) {
    try {
      const checkRes = await fetch(`${cmsApi}/api/pricing-plans?where[slug][equals]=${encodeURIComponent(pkg.slug)}`);
      if (checkRes.ok) {
        const checkJson = await checkRes.json();
        const payloadDoc = {
          name: pkg.name,
          slug: pkg.slug,
          tagline: pkg.tagline,
          monthlyPriceUsd: pkg.monthlyPriceUsd,
          yearlyPriceUsd: pkg.yearlyPriceUsd,
          badge: pkg.badge,
          highlighted: !!pkg.highlighted,
          activeOffers: pkg.activeOffers,
          flashSalesPerMonth: pkg.flashSalesPerMonth,
          teamSeats: pkg.teamSeats,
          features: pkg.features || [],
          excludedFeatures: (pkg.excludedFeatures || []).map((f: string) => ({ label: f })),
          sortOrder: pkg.sortOrder,
        };

        if (checkJson.totalDocs > 0) {
          const docId = checkJson.docs[0].id;
          await fetch(`${cmsApi}/api/pricing-plans/${docId}`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payloadDoc),
          });
          console.log(`[Sync Bridge] Updated CMS pricing plan: ${pkg.name}`);
        } else {
          await fetch(`${cmsApi}/api/pricing-plans`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payloadDoc),
          });
          console.log(`[Sync Bridge] Created CMS pricing plan: ${pkg.name}`);
        }
      }
    } catch (e) {
      console.warn(`[Sync Bridge] Pricing plan sync warning for ${pkg.slug}:`, e);
    }
  }

  console.log('[Sync Bridge] Synchronization complete.');
}
