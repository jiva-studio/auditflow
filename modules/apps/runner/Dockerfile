# syntax=docker/dockerfile:1.4

# ==============================================================================
# STAGE 1: COMPILATION & STATIC BINARY GENERATION
# ==============================================================================

FROM golang:1.24-alpine AS builder

WORKDIR /workspace

COPY modules/libs/ modules/libs/
COPY modules/apps/agent/ modules/apps/agent/
COPY modules/apps/streamer/ modules/apps/streamer/
COPY modules/apps/runner/ modules/apps/runner/

WORKDIR /workspace/modules/apps/agent
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /bin/agent ./cmd/main.go

WORKDIR /workspace/modules/apps/streamer
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /bin/streamer ./cmd/main.go

WORKDIR /workspace/modules/apps/runner
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /bin/runner ./cmd/main.go

# ==============================================================================
# STAGE 2: MINIMAL HARDENED RUNTIME
# ==============================================================================

FROM alpine:3.21 AS runner

RUN apk --no-cache add ca-certificates tzdata wget && \
    adduser -D -g '' -u 10001 -h /home/appuser appuser

USER 10001:10001

COPY --from=builder --chown=10001:10001 /bin/agent /bin/agent
COPY --from=builder --chown=10001:10001 /bin/streamer /bin/streamer
COPY --from=builder --chown=10001:10001 /bin/runner /bin/runner

ENV AGENT_BIN=/bin/agent
ENV STREAMER_BIN=/bin/streamer
ENV RULES_PATH=/rules.json
ENV DATA_DIR=/data
ENV SERVER_URL=http://server:8080

ENTRYPOINT ["/bin/runner"]
