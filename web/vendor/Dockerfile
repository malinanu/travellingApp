# Built with context = web/ (the npm workspace root), since this app depends
# on local @spotto/shared and @spotto/charts packages:
#   docker build -f web/vendor/Dockerfile --build-arg NEXT_PUBLIC_API_URL=... web/

FROM node:20-alpine AS builder
# NEXT_PUBLIC_* build args are Next.js's own convention for values it inlines
# into the client bundle, so this one is already public by design — fine to
# persist in image layer history. Never repurpose this ARG/ENV pair for an
# actual secret; use a runtime env var (not a build arg) for anything private.
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}
WORKDIR /web
COPY . .
RUN npm install
RUN npm run build -w vendor

# outputFileTracingRoot (set in vendor/next.config.mjs) pins the standalone
# output to this predictable layout: vendor/.next/standalone/vendor/server.js
FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
# node:20-alpine ships a built-in unprivileged `node` user (uid 1000) —
# the standard Next.js-recommended non-root runtime user.
COPY --from=builder --chown=node:node /web/vendor/.next/standalone ./
COPY --from=builder --chown=node:node /web/vendor/.next/static ./vendor/.next/static
# No public/ directory exists yet in this app — add a
# `COPY --from=builder --chown=node:node /web/vendor/public ./vendor/public` line back if one is added.
USER node
EXPOSE 3000
CMD ["node", "vendor/server.js"]
