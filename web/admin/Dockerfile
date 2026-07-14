# Built with context = web/ (the npm workspace root), since this app depends
# on the local @spotto/shared package:
#   docker build -f web/admin/Dockerfile --build-arg NEXT_PUBLIC_API_URL=... web/

FROM node:20-alpine AS builder
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}
WORKDIR /web
COPY . .
RUN npm install
RUN npm run build -w admin

# outputFileTracingRoot (set in admin/next.config.mjs) pins the standalone
# output to this predictable layout: admin/.next/standalone/admin/server.js
FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /web/admin/.next/standalone ./
COPY --from=builder /web/admin/.next/static ./admin/.next/static
# No public/ directory exists yet in this app — add a
# `COPY --from=builder /web/admin/public ./admin/public` line back if one is added.
EXPOSE 3001
CMD ["node", "admin/server.js"]
