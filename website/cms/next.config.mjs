import { withPayload } from '@payloadcms/next/withPayload';

/** @type {import('next').NextConfig} */
const nextConfig = {
  // Enables Payload CMS with Next.js App Router
  reactStrictMode: true,
};

export default withPayload(nextConfig);
