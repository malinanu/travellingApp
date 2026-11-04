'use client';

// Growth tools have been consolidated into the unified Promotions Hub (/offers).
// This client page redirects any legacy bookmarks or links to /offers?tab=flash.

import { useRouter } from 'next/navigation';
import { useEffect } from 'react';

export default function GrowthRedirectPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace('/offers?tab=flash');
  }, [router]);

  return (
    <div className="min-h-[50vh] flex items-center justify-center text-muted font-bold text-[14px]">
      Redirecting to Promotions Hub…
    </div>
  );
}
