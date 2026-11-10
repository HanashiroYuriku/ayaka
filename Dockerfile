# syntax=docker/dockerfile:1

# ==========================================
# STAGE 1: BUILD
# ==========================================
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Download dependencies first to leverage the layer cache.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# Static, stripped binary. Migrations and Swagger docs are embedded.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ayaka .

# ==========================================
# STAGE 2: RUNTIME
# ==========================================
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S ayaka && adduser -S -G ayaka -H ayaka

WORKDIR /app
COPY --from=builder /out/ayaka /app/ayaka
COPY config.yaml /app/config.yaml

# Never run as root.
USER ayaka

EXPOSE 8000

HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${SERVER_PORT:-8000}/livez" >/dev/null || exit 1

ENTRYPOINT ["/app/ayaka"]
CMD ["serve"]
