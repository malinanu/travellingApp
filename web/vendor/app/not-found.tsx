'use client';

import Link from 'next/link';

export default function NotFound() {
  return (
    <div className="min-h-screen bg-sand flex flex-col items-center justify-center p-6 text-center">
      <div className="w-16 h-16 rounded-full bg-accent-tint flex items-center justify-center mb-4 text-[24px]">
        🏝️
      </div>
      <h1 className="font-display font-extrabold text-[32px] text-ink mb-2">404 — Page Not Found</h1>
      <p className="text-[15px] font-semibold text-muted max-w-md mb-6">
        The page you are looking for does not exist or has been moved.
      </p>
      <Link
        href="/dashboard"
        className="btn-accent rounded-2xl px-6 py-3.5 font-extrabold text-[15px] inline-flex items-center gap-2"
      >
        Back to Dashboard
      </Link>
    </div>
  );
}
