# PAT Watchdog — independent monitoring + self-healing engine.
# Build with the Go toolchain, ship inside the docker:cli image so the
# watchdog can drive `docker`/`docker compose` against the host daemon
# (docker.sock is mounted at runtime). Pure-stdlib binary — no deps.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /watchdog .

FROM docker:27-cli
# CA certs for the ntfy HTTP call come with the alpine base.
COPY --from=build /watchdog /usr/local/bin/watchdog
ENTRYPOINT ["watchdog"]