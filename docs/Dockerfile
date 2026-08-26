# ─── Predict-A-Trade Documentation Container ───
# Self-hosted GitBook/Mintlify-like docs platform using Docsify
# No Node.js build step — serves markdown directly via nginx + docsify SPA

FROM nginx:alpine

# Copy Docsify entrypoint + docs content
COPY index.html /usr/share/nginx/html/index.html
COPY _sidebar.md /usr/share/nginx/html/_sidebar.md
COPY architecture/ /usr/share/nginx/html/architecture/
COPY strategy/ /usr/share/nginx/html/strategy/
COPY api/ /usr/share/nginx/html/api/
COPY database/ /usr/share/nginx/html/database/
COPY operations/ /usr/share/nginx/html/operations/
COPY guides/ /usr/share/nginx/html/guides/
COPY reports/ /usr/share/nginx/html/reports/
COPY README.md /usr/share/nginx/html/README.md

# Default nginx config (overridden by docker-compose volume mount)
RUN echo 'server { listen 2084; root /usr/share/nginx/html; index index.html; location / { try_files $uri $uri/ /index.html; } }' \
    > /etc/nginx/conf.d/default.conf

EXPOSE 2084
CMD ["nginx", "-g", "daemon off;"]
