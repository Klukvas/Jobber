# Multi-stage build for React frontend
# Build stage — debian-slim so Playwright's Chromium has its libc/X deps.
FROM node:20-bookworm-slim AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install npm deps + Playwright Chromium (used by scripts/prerender.mjs).
# --with-deps pulls in the system libs Chromium needs (libnss3, libatk, fonts, ...).
RUN npm install && npx playwright install --with-deps chromium

# Build arguments for Vite env vars (read at npm run build time via ARG)
# ARG-only: values are NOT persisted in the final image (no ENV promotion)
ARG VITE_API_BASE_URL=/api/v1
ARG VITE_FEATURE_SENTRY=false
ARG VITE_FEATURE_PAYMENTS=false
ARG VITE_FEATURE_EMAIL_NOTIFICATIONS=true
ARG VITE_SENTRY_DSN=
# Toggle prerender (default on). Pass --build-arg PRERENDER=false to skip.
ARG PRERENDER=true

# Copy source code
COPY . .

# Build + prerender public routes to static HTML for SEO.
RUN if [ "$PRERENDER" = "true" ]; then npm run build:prerender; else npm run build; fi

# Runtime stage - serve with nginx
FROM nginx:alpine

# Copy custom nginx config
COPY --from=builder /app/dist /usr/share/nginx/html

# Create nginx config for SPA with API proxy
RUN echo 'server { \
    listen 80; \
    server_name _; \
    root /usr/share/nginx/html; \
    index index.html; \
    \
    # Security headers \
    add_header X-Frame-Options "SAMEORIGIN" always; \
    add_header X-Content-Type-Options "nosniff" always; \
    add_header X-XSS-Protection "1; mode=block" always; \
    add_header Referrer-Policy "strict-origin-when-cross-origin" always; \
    add_header Permissions-Policy "geolocation=(), microphone=(), camera=(), payment=(), usb=()" always; \
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always; \
    \
    # Gzip compression \
    gzip on; \
    gzip_vary on; \
    gzip_min_length 1024; \
    gzip_comp_level 6; \
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/xml application/xml+rss application/json application/ld+json image/svg+xml; \
    \
    # Proxy API requests to backend \
    location /api/ { \
        proxy_pass http://backend:8080; \
        proxy_http_version 1.1; \
        proxy_set_header Host $host; \
        proxy_set_header X-Real-IP $remote_addr; \
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; \
        proxy_set_header X-Forwarded-Proto $scheme; \
        proxy_connect_timeout 30s; \
        proxy_send_timeout 30s; \
        proxy_read_timeout 30s; \
    } \
    \
    # HTML — never cache (SPA entry point changes every deploy) \
    location ~* \.html$ { \
        add_header Cache-Control "no-cache, no-store, must-revalidate"; \
        add_header Pragma "no-cache"; \
        add_header Expires "0"; \
    } \
    \
    # SEO feeds (sitemap.xml, robots.txt, llms.txt) — short cache so updates propagate \
    location ~* \.(xml|txt)$ { \
        expires 1h; \
        add_header Cache-Control "public, must-revalidate"; \
    } \
    \
    # Favicons — short cache so updates propagate quickly \
    location ~* ^/(favicon\.|apple-touch-icon) { \
        expires 1h; \
        add_header Cache-Control "public, must-revalidate"; \
    } \
    \
    # Hashed static assets (Vite adds content hash) — cache forever \
    location ~* \.(js|css|woff|woff2|ttf|eot)$ { \
        expires 1y; \
        add_header Cache-Control "public, immutable"; \
    } \
    \
    # Images — moderate cache \
    location ~* \.(png|jpg|jpeg|gif|ico|svg)$ { \
        expires 7d; \
        add_header Cache-Control "public"; \
    } \
    \
    # SPA routing - all routes go to index.html \
    location / { \
        try_files $uri $uri/ /index.html; \
        add_header Cache-Control "no-cache, no-store, must-revalidate"; \
    } \
}' > /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
