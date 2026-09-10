# Predict-A-Trade — Discord Bot (alerts · portfolio · price prediction)
FROM node:22-alpine
WORKDIR /app
RUN apk add --no-cache ca-certificates curl
COPY discord-bot/package.json ./
RUN npm install --omit=dev
COPY discord-bot/bot.mjs ./
ENV NODE_ENV=production
# No HTTP port — the bot is a Discord client (gateway WS + polling)
HEALTHCHECK --interval=30s --timeout=8s --retries=5 \
  CMD node -e "fetch('https://discord.com/api/v10/gateway',{method:'HEAD'}).then(()=>process.exit(0)).catch(()=>process.exit(1))" || exit 1
CMD ["node", "bot.mjs"]