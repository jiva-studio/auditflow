# syntax=docker/dockerfile:1.4

# ==============================================================================
# STAGE 1: COMPILATION & STATIC BINARY GENERATION
# ==============================================================================

FROM golang:1.24-alpine AS builder

WORKDIR /workspace

COPY modules/libs/domain/ modules/libs/domain/
COPY modules/libs/protocol/ modules/libs/protocol/
COPY modules/libs/rules/ modules/libs/rules/
COPY modules/apps/agent/ modules/apps/agent/

WORKDIR /workspace/modules/apps/agent
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /bin/agent ./cmd/main.go

# ==============================================================================
# STAGE 2: MINIMAL HARDENED RUNTIME
# ==============================================================================

FROM alpine:3.21 AS runner

RUN apk --no-cache add ca-certificates tzdata wget && \
    adduser -D -g '' -u 10001 -h /home/appuser appuser

USER 10001:10001

COPY --from=builder --chown=10001:10001 /bin/agent /bin/agent

EXPOSE 8081
ENTRYPOINT ["/bin/agent"]
