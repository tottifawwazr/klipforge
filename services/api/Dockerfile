# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.24

FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -buildvcs=false \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/klipforge-api \
    ./cmd/api

FROM alpine:3.21 AS runtime

RUN apk add --no-cache ca-certificates \
    && addgroup -S klipforge \
    && adduser -S -G klipforge klipforge

COPY --from=build /out/klipforge-api /usr/local/bin/klipforge-api

USER klipforge

ENV HTTP_HOST=0.0.0.0 \
    HTTP_PORT=8080

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${HTTP_PORT:-8080}/api/v1/health" || exit 1

ENTRYPOINT ["/usr/local/bin/klipforge-api"]
