# ── Stage 1: Build ───────────────────────────────────────────────────────────
FROM node:lts-alpine AS builder

WORKDIR /build

# Copy manifests first for layer caching.
# npm ci only re-runs when package-lock.json changes.
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# ── Stage 2: Serve ───────────────────────────────────────────────────────────
FROM nginx:alpine

COPY --from=builder /build/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
