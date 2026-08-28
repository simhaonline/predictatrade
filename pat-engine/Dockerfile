# pat-engine — minimal distroless-ish Go image.
FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/gateway ./cmd/gateway && \
    CGO_ENABLED=0 go build -o /out/agent ./cmd/agent && \
    CGO_ENABLED=0 go build -o /out/sim ./cmd/sim

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/gateway /app/gateway
COPY --from=build /out/agent /app/agent
COPY --from=build /out/sim /app/sim
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/app/gateway"]
