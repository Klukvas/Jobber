# Multi-stage build for React frontend
# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install

# Build arguments for Vite env vars (read at npm run build time via ARG)
# ARG-only: values are NOT persisted in the final image (no ENV promotion)
ARG VITE_API_BASE_URL=/api/v1
ARG VITE_FEATURE_SENTRY=false
ARG VITE_FEATURE_PAYMENTS=false
ARG VITE_FEATURE_EMAIL_NOTIFICATIONS=true
ARG VITE_SENTRY_DSN=

# Copy source code
COPY . .

# Build the application
RUN npm run build

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
    \
    # Gzip compression \
    gzip on; \
    gzip_vary on; \
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/xml+rss application/json; \
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
