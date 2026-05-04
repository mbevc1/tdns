.PHONY: help vendor build clean clean-vendor clean-cache tidy run snapshot dev test fmt fmt-check lint vet
.DEFAULT: help
ifndef VERBOSE
.SILENT:
endif

VER?=dev
GHASH:=$(shell git rev-parse --short HEAD)
VERSION?=$(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo v0)
GOTELEMETRY:=	off
GO:=            go
#GO_BUILD:=      go build -mod vendor -ldflags "-s -w -X main.GitCommit=${GHASH} -X main.Version=${VERSION}"
GO_BUILD:=      go build -mod mod -ldflags "-s -w -X main.commit=${GHASH} -X main.version=${VERSION}"
#VERSION="${VERSION}" goreleaser --snapshot --rm-dist
GO_VENDOR:=     go mod vendor
BIN:=           tdns

#help: ## Show this help
#	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[33m%-20s\033[0m %s\n", $$1, $$2}'
help: ## Show help for each Makefile target
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "🎯 \033[36m%-15s\033[0m %s\n", $$1, $$2}'

#$(BIN): vendor ## Produce binary
$(BIN): ## Produce binary
	@echo "🔨 Building $(BINARY_NAME)..."
	GO111MODULE=on $(GO_BUILD)
	@echo "✅ Built binary $(BIN)"
	upx $@

vendor: **/*.go ## Build vendor deps
	GO111MODULE=on $(GO_VENDOR)

clean: clean-vendor ## Clean artefacts
	@echo "🧹 Cleaning up..."
	rm -rf $(BIN) $(BIN)_* $(BIN).exe dist/
	@echo "✅ Clean complete"

clean-vendor: ## Clean vendor folder
	rm -rf vendor

clean-cache: ## Clean Go module cache
	@echo "🧼 Cleaning Go module cache..."
	@go clean -modcache
	@echo "✅ Module cache cleaned"

build: clean $(BIN) ## Build binary
	#upx $(BIN)

tidy: ## Clean up go.mod and go.sum
	@echo "🧼 Tidying go.mod and go.sum..."
	@go mod tidy
	@echo "✅ Done."

run:
	go run .

snapshot: clean ## Build local snapshot
	goreleaser build --clean --snapshot --single-target

dev: clean ## Dev test target
	go build -ldflags "-s -w -X main.version=${VER}" -o $(BIN)
	upx $@

test: vendor ## Run tests
	go test -v ./...

fmt: **/*.go ## Format Golang code
	go fmt ./...

fmt-check: ## Fail if Go code is not gofmt-clean
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not gofmt-clean:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

lint:
	golint ./...

vet:
	go vet -all ./...
