# ==============================================================================
# AGENT E2E TEST MAKEFILE
# ==============================================================================

ROOT_DIR := $(abspath ../../..)
BIN_DIR := $(ROOT_DIR)/bin
STREAMER_BIN := $(BIN_DIR)/streamer
AGENT_BIN := $(BIN_DIR)/agent

.PHONY: all build test clean

all: test

build:
	@mkdir -p $(BIN_DIR)
	cd $(ROOT_DIR)/modules/apps/streamer && go build -o $(STREAMER_BIN) ./cmd/main.go
	cd $(ROOT_DIR)/modules/apps/agent && go build -o $(AGENT_BIN) ./cmd/main.go

test: build
	STREAMER_BIN=$(STREAMER_BIN) AGENT_BIN=$(AGENT_BIN) DATA_DIR=$(ROOT_DIR)/data RULES_PATH=$(ROOT_DIR)/data/rules.json go test -v -count=1 ./...

clean:
	rm -f $(STREAMER_BIN) $(AGENT_BIN)
