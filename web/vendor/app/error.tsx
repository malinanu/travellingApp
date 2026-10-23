'use client';

import { useEffect } from 'react';

// A ChunkLoadError means this tab is running code from an earlier deploy:
// each deploy replaces /_next/static, so a lazily-loaded file the old code
// asks for (the map on the registration wizard's last step, for one) is
// gone. reset() would only fail again, so reload to pick up the current
// build: once automatically, then via the button.
const CHUNK_RELOAD_KEY = 'spotto_vendor_chunk_reload';

function isChunkLoadError(error: Error) {
  return error.name === 'ChunkLoadError' || /Failed to load chunk|Loading chunk/i.test(error.message);
}

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const staleDeploy = isChunkLoadError(error);

  useEffect(() => {
    console.error('Vendor portal error:', error);
    if (!staleDeploy) return;
    try {
      // Guard against a reload loop if the file is missing for some other
      // reason: only auto-reload if we haven't just done so.
      const last = Number(window.sessionStorage.getItem(CHUNK_RELOAD_KEY) ?? 0);
      if (Date.now() - last < 30_000) return;
      window.sessionStorage.setItem(CHUNK_RELOAD_KEY, String(Date.now()));
      window.location.reload();
    } catch {
      /* storage blocked: the Reload button below still works */
    }
  }, [error, staleDeploy]);

  return (
    <div className="min-h-screen bg-sand flex flex-col items-center justify-center p-6 text-center">
      <div className="w-16 h-16 rounded-full bg-coral/10 flex items-center justify-center mb-4 text-[24px]">
        ⚠️
      </div>
      <h1 className="font-display font-extrabold text-[32px] text-ink mb-2">
        {staleDeploy ? 'Spotto was just updated' : '500 — Something Went Wrong'}
      </h1>
      <p className="text-[15px] font-semibold text-muted max-w-md mb-6">
        {staleDeploy
          ? 'This page is from the previous version. Reload to continue.'
          : 'An unexpected error occurred. Please try again.'}
      </p>
      <button
        onClick={() => (staleDeploy ? window.location.reload() : reset())}
        className="btn-accent rounded-2xl px-6 py-3.5 font-extrabold text-[15px] inline-flex items-center gap-2"
      >
        {staleDeploy ? 'Reload' : 'Try Again'}
      </button>
    </div>
  );
}
