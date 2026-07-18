import path from 'node:path';

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  reactStrictMode: false,
  // Pin the trace root to the workspace root (web/) so the standalone
  // output has a predictable layout for the Dockerfile to copy — otherwise
  // Next.js's monorepo auto-detection can vary the nesting. (Top-level as
  // of Next 16 — it lived under `experimental` in 14.2.)
  outputFileTracingRoot: path.join(process.cwd(), '..'),
  transpilePackages: ['@spotto/shared', '@spotto/charts'],
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          { key: 'X-Frame-Options', value: 'DENY' },
          { key: 'Content-Security-Policy', value: "frame-ancestors 'none'" },
          { key: 'X-Content-Type-Options', value: 'nosniff' },
          { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
        ],
      },
    ];
  },
};

export default nextConfig;
