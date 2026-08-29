# pat-mail — send-only submission relay (predictatrade mail plane)
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /pat-mail .
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /pat-mail /usr/local/bin/pat-mail
ENV SQLITE_PATH=/var/lib/pat-mail/spool.db
EXPOSE 587 465
VOLUME ["/var/lib/pat-mail"]
ENTRYPOINT ["/usr/local/bin/pat-mail"]
