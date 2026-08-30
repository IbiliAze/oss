SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

# ---- versions (bump deliberately, never float) ----
BUF_VERSION              := v1.47.2
PROTOC_GEN_GO_VERSION    := v1.36.4
PROTOC_GEN_GO_GRPC_VERSION := v1.5.1

BIN        := $(CURDIR)/bin
PROTO_DIR  := api/proto
GEN_DIR    := api/gen
export PATH := $(BIN):$(PATH)

BUF               := $(BIN)/buf
PROTOC_GEN_GO     := $(BIN)/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(BIN)/protoc-gen-go-grpc

.PHONY: help
help: ## List targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n",$$1,$$2}'

# ---- tooling ----
$(BIN):
	@mkdir -p $(BIN)

$(BUF): | $(BIN)
	GOBIN=$(BIN) go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)

$(PROTOC_GEN_GO): | $(BIN)
	GOBIN=$(BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)

$(PROTOC_GEN_GO_GRPC): | $(BIN)
	GOBIN=$(BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)

.PHONY: tools
tools: $(BUF) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) ## Install pinned codegen tools into ./bin

# ---- codegen ----
.PHONY: proto
proto: tools ## Regenerate Go from .proto (wipes gen/ first to clear stale files)
	rm -rf $(GEN_DIR)
	$(BUF) generate
	go mod tidy

.PHONY: proto-lint
proto-lint: $(BUF) ## Lint .proto files
	$(BUF) lint

.PHONY: proto-format
proto-format: $(BUF) ## Format .proto files in place
	$(BUF) format -w

.PHONY: proto-breaking
proto-breaking: $(BUF) ## Detect breaking changes against origin/main
	$(BUF) breaking --against '.git#branch=origin/main,subdir=.'

.PHONY: proto-verify
proto-verify: proto ## CI gate: fail if generated code is out of date
	@git diff --exit-code -- $(GEN_DIR) \
		|| { echo "gen/ is stale — run 'make proto' and commit"; exit 1; }

.PHONY: proto-clean
proto-clean: ## Remove generated code and local tooling
	rm -rf $(GEN_DIR) $(BIN)