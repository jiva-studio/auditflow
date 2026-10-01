# ==============================================================================
# CONFIGURATION & UTILITIES
# ==============================================================================

MUTATE_CMD ?= go run github.com/go-gremlins/gremlins/cmd/gremlins@latest unleash

.PHONY: fmt \
        domain_guard domain_test domain_lint domain_mutate \
        protocol_gen protocol_test protocol_lint \
        rules_test rules_lint rules_mutate \
        agent_test agent_lint agent_mutate \
        server_test server_lint server_mutate \
        streamer_test streamer_lint streamer_mutate \
        e2e_test

fmt:
	gofmt -s -w modules/ tests/

# ==============================================================================
# LIBS / DOMAIN
# ==============================================================================

domain_guard:
	python3 scripts/guards/domain/no_json.py
	python3 scripts/guards/domain/no_io.py

domain_test:
	cd modules/libs/domain && go test -count=1 ./...

domain_lint:
	cd modules/libs/domain && golangci-lint run ./...

domain_mutate:
	cd modules/libs/domain && $(MUTATE_CMD)

# ==============================================================================
# LIBS / PROTOCOL
# ==============================================================================

protocol_gen:
	protoc -I modules/libs/protocol/proto/v1 --go_out=modules/libs/protocol/gen/go/v1 --go_opt=paths=source_relative modules/libs/protocol/proto/v1/events.proto modules/libs/protocol/proto/v1/audit.proto

protocol_test:
	cd modules/libs/protocol && go test -count=1 ./...

protocol_lint:
	cd modules/libs/protocol && golangci-lint run ./...

# ==============================================================================
# LIBS / RULES
# ==============================================================================

rules_test:
	cd modules/libs/rules && go test -count=1 ./...

rules_lint:
	cd modules/libs/rules && golangci-lint run ./...

rules_mutate:
	cd modules/libs/rules && $(MUTATE_CMD)


# ==============================================================================
# APPS / AGENT
# ==============================================================================

agent_test:
	cd modules/apps/agent && go test -count=1 ./...

agent_lint:
	cd modules/apps/agent && golangci-lint run ./...

agent_mutate:
	cd modules/apps/agent && $(MUTATE_CMD)

# ==============================================================================
# APPS / SERVER
# ==============================================================================

server_test:
	cd modules/apps/server && go test -count=1 ./...

server_lint:
	cd modules/apps/server && golangci-lint run ./...

server_mutate:
	cd modules/apps/server && $(MUTATE_CMD)

# ==============================================================================
# APPS / STREAMER
# ==============================================================================

streamer_test:
	cd modules/apps/streamer && go test -count=1 ./...

streamer_lint:
	cd modules/apps/streamer && golangci-lint run ./...

streamer_mutate:
	cd modules/apps/streamer && $(MUTATE_CMD)

# ==============================================================================
# TESTS / E2E
# ==============================================================================

streamer_e2e_test:
	$(MAKE) -C tests/e2e/streamer test

agent_e2e_test:
	$(MAKE) -C tests/e2e/agent test

server_e2e_test:
	$(MAKE) -C tests/e2e/server test

e2e_test: streamer_e2e_test agent_e2e_test server_e2e_test

