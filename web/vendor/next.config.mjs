import path from 'node:path';

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  // Pin the trace root to the workspace root (web/) so the standalone
  // output has a predictable layout for the Dockerfile to copy — otherwise
  // Next.js's monorepo auto-detection can vary the nesting. (Next 14.2
  // still expects this under `experimental`.)
  experimental: {
    outputFileTracingRoot: path.join(process.cwd(), '..'),
  },
  transpilePackages: ['@spotto/shared', '@spotto/charts'],
};

export default nextConfig;
