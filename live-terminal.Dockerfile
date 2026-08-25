# Predict-A-Trade — Live Terminal (public edge: static terminal + preview funnel + proxy)
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY realtime/go.mod realtime/go.sum ./
RUN go mod download
COPY realtime/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/live-terminal ./cmd/live-terminal

FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=builder /out/live-terminal ./live-terminal
COPY live-dashboard/ ./public/
ENV LIVE_TERMINAL_PORT=13090 \
    LIVE_TERMINAL_STATIC_DIR=/app/public
EXPOSE 13090
HEALTHCHECK --interval=10s --timeout=5s --retries=10 \
  CMD curl --fail --silent http://127.0.0.1:13090/health || exit 1
CMD ["./live-terminal"]
