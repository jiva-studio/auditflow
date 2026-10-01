# ==============================================================================
# CONFIGURATION & UTILITIES
# ==============================================================================

MUTATE_CMD ?= go run github.com/go-gremlins/gremlins/cmd/gremlins@latest unleash

.PHONY: fmt \
        domain_guard domain_test domain_lint domain_mutate \
        agent_test agent_lint agent_mutate \
        server_test server_lint server_mutate \
        streamer_test streamer_lint streamer_mutate

fmt:
	gofmt -s -w libs/ modules/ tests/

# ==============================================================================
# LIBS / DOMAIN
# ==============================================================================

domain_guard:
	python3 scripts/guards/domain/no_json.py
	python3 scripts/guards/domain/no_io.py

domain_test:
	cd libs/domain && go test -count=1 ./...

domain_lint:
	cd libs/domain && golangci-lint run ./...

domain_mutate:
	cd libs/domain && $(MUTATE_CMD)

# ==============================================================================
# MODULES / AGENT
# ==============================================================================

agent_test:
	cd modules/agent && go test -count=1 ./...

agent_lint:
	cd modules/agent && golangci-lint run ./...

agent_mutate:
	cd modules/agent && $(MUTATE_CMD)

# ==============================================================================
# MODULES / SERVER
# ==============================================================================

server_test:
	cd modules/server && go test -count=1 ./...

server_lint:
	cd modules/server && golangci-lint run ./...

server_mutate:
	cd modules/server && $(MUTATE_CMD)

# ==============================================================================
# MODULES / STREAMER
# ==============================================================================

streamer_test:
	cd modules/streamer && go test -count=1 ./...

streamer_lint:
	cd modules/streamer && golangci-lint run ./...

streamer_mutate:
	cd modules/streamer && $(MUTATE_CMD)
