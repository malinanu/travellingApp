'use client';

import { useEffect } from 'react';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('Admin Command Center error:', error);
  }, [error]);

  return (
    <div className="min-h-screen bg-sand flex flex-col items-center justify-center p-6 text-center">
      <div className="w-16 h-16 rounded-full bg-coral/10 flex items-center justify-center mb-4 text-[24px]">
        🚨
      </div>
      <h1 className="font-display font-extrabold text-[32px] text-ink mb-2">500 — Command Center Error</h1>
      <p className="text-[15px] font-semibold text-muted max-w-md mb-6">
        An internal admin error occurred.
      </p>
      <button
        onClick={() => reset()}
        className="btn-accent rounded-2xl px-6 py-3.5 font-extrabold text-[15px] inline-flex items-center gap-2"
      >
        Reload Page
      </button>
    </div>
  );
}
