# Predict-A-Trade — Status Page (plain Node.js, no dependencies)
FROM node:24-alpine
WORKDIR /app
COPY server.js ./
ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=13083
EXPOSE 13083
CMD ["node", "server.js"]
