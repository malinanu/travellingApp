# Built with context = web/ (the npm workspace root), since this app depends
# on local @spotto/shared and @spotto/charts packages:
#   docker build -f web/vendor/Dockerfile --build-arg NEXT_PUBLIC_API_URL=... web/

FROM node:20-alpine AS builder
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
COPY --from=builder /web/vendor/.next/standalone ./
COPY --from=builder /web/vendor/.next/static ./vendor/.next/static
# No public/ directory exists yet in this app — add a
# `COPY --from=builder /web/vendor/public ./vendor/public` line back if one is added.
EXPOSE 3000
CMD ["node", "vendor/server.js"]
