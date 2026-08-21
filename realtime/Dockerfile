# Predict-A-Trade — Go Real-Time Engine
# Multi-stage build: build stage + runtime stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o bin/realtime-engine ./cmd/realtime-engine/

# Runtime stage — minimal image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=UTC

WORKDIR /app
COPY --from=builder /app/bin/realtime-engine /app/realtime-engine

EXPOSE 13081
CMD ["/app/realtime-engine"]
