# Predict-A-Trade — Go Real-Time Engine
FROM golang:1.25 AS builder

RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Clear Go build cache to ensure fresh compile with latest source files
RUN go clean -cache
RUN CGO_ENABLED=1 go build -o bin/realtime-engine ./cmd/realtime-engine/

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl tzdata && rm -rf /var/lib/apt/lists/*
ENV TZ=UTC

WORKDIR /app
COPY --from=builder /app/bin/realtime-engine /app/realtime-engine
# Ship the research-trained calibration models into the runtime image so the
# engine loads empirically-calibrated probabilities at startup (the Live
# Calibrator also writes live retrains here at runtime).
COPY --from=builder /app/calibration /app/calibration

EXPOSE 13081
CMD ["/app/realtime-engine"]
