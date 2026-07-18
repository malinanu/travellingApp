'use client';

// Shared error/loading building blocks for admin routes — one plain,
// consistent look reused by every route's error.tsx/loading.tsx rather
// than bespoke per-route content.

import { AlertTriangle, RotateCw } from 'lucide-react';

export function RouteError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <div className="min-h-screen flex items-center justify-center" style={{ background: 'var(--bg)' }}>
      <div className="panel p-8 max-w-sm text-center">
        <AlertTriangle size={28} className="mx-auto mb-3" style={{ color: '#e11d48' }} />
        <p className="font-bold text-[15px] mb-1.5">Something went wrong.</p>
        <p className="text-[13px] text-muted font-medium mb-5">{error.message || 'This page failed to load.'}</p>
        <button className="btn btn-primary flex items-center gap-2 mx-auto" onClick={reset}>
          <RotateCw size={14} /> Try again
        </button>
      </div>
    </div>
  );
}

export function RouteLoading() {
  return (
    <div className="min-h-screen flex items-center justify-center text-muted font-bold" style={{ background: 'var(--bg)' }}>
      Loading…
    </div>
  );
}

/** Inline banner a page renders near its query results when isError is
 * true — react-query swallows fetch errors by default (no throwOnError),
 * so this is what actually surfaces a failed GET, distinct from
 * error.tsx (which only ever catches a render-time throw). */
export function InlineErrorBanner({ message }: { message?: string }) {
  return (
    <div className="rounded-lg px-4 py-3 mb-4 flex items-center gap-2.5" style={{ background: 'rgba(225,29,72,0.08)', color: '#e11d48' }}>
      <AlertTriangle size={15} className="shrink-0" />
      <p className="font-semibold text-[13px]">{message ?? "Couldn't load — try refreshing."}</p>
    </div>
  );
}
