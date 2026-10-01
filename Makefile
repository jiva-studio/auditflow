# ==============================================================================
# GLOBAL UTILITIES
# ==============================================================================

.PHONY: fmt \
        domain_test domain_lint \
        agent_test agent_lint \
        server_test server_lint \
        streamer_test streamer_lint \
        e2e_test

fmt:
	gofmt -s -w libs/ modules/ tests/

# ==============================================================================
# LIBS / DOMAIN
# ==============================================================================

domain_test:
	cd libs/domain && go test -count=1 ./...

domain_lint:
	cd libs/domain && golangci-lint run ./...

# ==============================================================================
# MODULES / AGENT
# ==============================================================================

agent_test:
	cd modules/agent && go test -count=1 ./...

agent_lint:
	cd modules/agent && golangci-lint run ./...

# ==============================================================================
# MODULES / SERVER
# ==============================================================================

server_test:
	cd modules/server && go test -count=1 ./...

server_lint:
	cd modules/server && golangci-lint run ./...

# ==============================================================================
# MODULES / STREAMER
# ==============================================================================

streamer_test:
	@if [ -f modules/streamer/go.mod ]; then cd modules/streamer && go test -count=1 ./...; fi

streamer_lint:
	@if [ -f modules/streamer/go.mod ]; then cd modules/streamer && golangci-lint run ./...; fi

# ==============================================================================
# TESTS / E2E
# ==============================================================================

e2e_test:
	@if [ -f tests/e2e/go.mod ]; then cd tests/e2e && go test -count=1 ./...; fi
