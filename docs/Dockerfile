# dumpsock docs container — multi-stage.
# Stage 1 builds both VitePress halves; stage 2 wraps the chosen variant
# in caddy:2-alpine. VARIANT build-arg picks {external, internal}.
# Same pattern as lab-hartle-tech + medusa + tools (see
# code-hartle-tech/hartle.tech-terraform k8s/workloads/product-docs/).

ARG VARIANT=external

# ── build stage ──────────────────────────────────────────────────────
FROM node:20-alpine AS build
WORKDIR /src

COPY package.json package-lock.json* ./
RUN npm ci --no-audit --no-fund

COPY . .
RUN npm run external:build && npm run internal:build

# ── runtime stage ────────────────────────────────────────────────────
FROM caddy:2-alpine AS runtime
ARG VARIANT
WORKDIR /srv

COPY --from=build /src/${VARIANT}/.vitepress/dist /srv
COPY Caddyfile /etc/caddy/Caddyfile

USER nobody
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=2s \
  CMD wget --quiet --spider http://127.0.0.1:8080/ || exit 1

CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
