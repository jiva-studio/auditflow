# syntax=docker/dockerfile:1.4

# ==============================================================================
# STAGE 1: DEPENDENCIES & CACHING
# ==============================================================================

FROM golang:1.24-alpine AS deps

WORKDIR /workspace

COPY libs/domain/go.mod libs/domain/
COPY libs/protocol/go.mod libs/protocol/go.sum* libs/protocol/
COPY modules/streamer/go.mod modules/streamer/go.sum* modules/streamer/

WORKDIR /workspace/modules/streamer
RUN go mod download

# ==============================================================================
# STAGE 2: COMPILATION & ARTIFACT GENERATION
# ==============================================================================

FROM deps AS builder

WORKDIR /workspace

COPY libs/domain libs/domain/
COPY libs/protocol libs/protocol/
COPY modules/streamer modules/streamer/

WORKDIR /workspace/modules/streamer
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /bin/streamer ./cmd/main.go

# ==============================================================================
# STAGE 3: MINIMAL HARDENED RUNTIME IMAGE
# ==============================================================================

FROM alpine:3.21 AS runner

RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -g '' -u 10001 -h /home/appuser appuser

USER 10001:10001

COPY --from=builder --chown=10001:10001 /bin/streamer /bin/streamer

ENTRYPOINT ["/bin/streamer"]
