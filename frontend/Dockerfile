# Predict-A-Trade — Next.js Frontend
FROM node:24-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install --legacy-peer-deps

FROM node:24-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# NEXT_PUBLIC_* vars are inlined into the client bundle at build time — they
# MUST be provided as build args (runtime env_file alone is NOT enough).
ARG NEXT_PUBLIC_API_BASE_URL=https://api.predictatrade.com/api/v1
ARG NEXT_PUBLIC_API_URL=https://api.predictatrade.com/api/v1
ARG NEXT_PUBLIC_WS_URL=wss://platform.predictatrade.com/ws/v1/relay
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_WS_URL=$NEXT_PUBLIC_WS_URL
RUN npx next build --webpack

FROM node:24-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
ENV PORT=13082
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json
EXPOSE 13082
CMD ["node", "node_modules/.bin/next", "start", "-H", "0.0.0.0", "-p", "13082"]
