# syntax=docker/dockerfile:1
FROM node:22.22.3-alpine@sha256:e58326d0d441090181ac150dc2078d3e2cf6a0d42e809aebba3ef5880935ffdd AS build

WORKDIR /app
COPY mise.toml ./
WORKDIR /app/site
COPY site/package.json site/pnpm-lock.yaml ./
ENV ASTRO_TELEMETRY_DISABLED=1 KB_SITE_BASE=/
RUN node --input-type=module -e 'import { readFileSync } from "node:fs"; import assert from "node:assert/strict"; assert.equal(process.versions.node, readFileSync("../mise.toml", "utf8").match(/^node = "([^"]+)"/m)[1]);' \
    && corepack enable \
    && pnpm install --frozen-lockfile

COPY site/astro.config.mjs site/tsconfig.json ./
COPY site/src/ ./src/
COPY site/public/ ./public/
COPY assets/logo.png /app/assets/logo.png
RUN pnpm run check && pnpm run build

FROM nginx:stable-alpine-slim@sha256:dede8895648b461178c3bc604f7eed4423efe5220b2c4d839f5eb3d34ac3f8f7 AS runtime

COPY --from=build /app/site/dist/ /usr/share/nginx/html/
RUN <<'SH'
cat > /etc/nginx/nginx.conf <<'NGINX'
worker_processes auto;
pid /tmp/nginx.pid;
error_log /dev/stderr warn;
events { worker_connections 1024; }
http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    access_log /dev/stdout;
    sendfile on;
    server_tokens off;
    client_body_temp_path /tmp/client_body;
    proxy_temp_path /tmp/proxy;
    fastcgi_temp_path /tmp/fastcgi;
    uwsgi_temp_path /tmp/uwsgi;
    scgi_temp_path /tmp/scgi;
    server {
        listen 8080;
        server_name _;
        root /usr/share/nginx/html;
        index index.html;
        location = /health {
            access_log off;
            default_type text/plain;
            return 200 "ok\n";
        }
        location /_astro/ {
            add_header Cache-Control "public, max-age=31536000, immutable";
            try_files $uri =404;
        }
        location / {
            add_header Cache-Control "no-cache";
            try_files $uri $uri/ =404;
        }
    }
}
NGINX
SH

USER nginx
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["nginx"]
CMD ["-g", "daemon off;"]
