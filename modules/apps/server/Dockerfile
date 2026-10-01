# syntax=docker/dockerfile:1.4

# ==============================================================================
# STAGE 1: COMPILATION & STATIC BINARY GENERATION
# ==============================================================================

FROM golang:1.24-alpine AS builder

WORKDIR /workspace

COPY modules/libs/domain/ modules/libs/domain/
COPY modules/libs/protocol/ modules/libs/protocol/
COPY modules/libs/telemetry/ modules/libs/telemetry/
COPY modules/apps/server/ modules/apps/server/

WORKDIR /workspace/modules/apps/server
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /bin/server ./cmd/main.go

# ==============================================================================
# STAGE 2: MINIMAL HARDENED RUNTIME
# ==============================================================================

FROM alpine:3.21 AS runner

RUN apk --no-cache add ca-certificates tzdata wget && \
    adduser -D -g '' -u 10001 -h /home/appuser appuser

USER 10001:10001

COPY --from=builder --chown=10001:10001 /bin/server /bin/server

EXPOSE 8080
ENTRYPOINT ["/bin/server"]
