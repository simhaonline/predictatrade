# Predict-A-Trade — Nginx Reverse Proxy
# Includes static live dashboard and SSL termination
FROM nginx:alpine

# Copy nginx configs
COPY nginx/sites-available/ /etc/nginx/sites-available/
COPY nginx/snippets/ /etc/nginx/snippets/
COPY nginx/nginx.conf /etc/nginx/nginx.conf

# Copy static live dashboard
COPY live-dashboard/ /var/www/pat-live/

# SSL certs will be mounted as volume
EXPOSE 80 443
CMD ["nginx", "-g", "daemon off;"]
