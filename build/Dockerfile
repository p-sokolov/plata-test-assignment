# syntax=docker/dockerfile:1.7
ARG GO_VERSION=1.26.3

FROM golang:${GO_VERSION}-alpine AS base

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

FROM base AS builder

COPY . .

ENV CGO_ENABLED=0 \
    GOCACHE=/root/.cache/go-build

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -trimpath -ldflags="-s -w" -o /service ./cmd/app

FROM base AS dev

ARG AIR_VERSION=v1.63.4

RUN --mount=type=cache,target=/go/pkg/mod \
    go install github.com/air-verse/air@${AIR_VERSION}

CMD ["air", "-c", "air.toml"]

FROM scratch AS runner

WORKDIR /app

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /service /app/service

USER 65532:65532

ENTRYPOINT ["/app/service"]
