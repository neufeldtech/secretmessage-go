M 	 = $(shell printf "\033[34;1m▶\033[0m")

PROJECT_NAME   ?= secretmessage
OS             := $(shell uname -s)
GOOS           ?= $(shell echo $(OS) | tr '[:upper:]' '[:lower:]')

BUILD_OPTS   = -gcflags="-trimpath=$(GOPATH)/src"

lint: ; $(info $(M) running linter…) @
	golangci-lint run -v

tidy: ; $(info $(M) Cleaning up dependencies…) @
	GOPRIVATE=github.com/contentsquare/* go mod tidy

deps: ; $(info $(M) Fetching dependencies…) @
	$(info $(PROJECT_NAME))
	go mod download

build: deps tidy ; $(info $(M) Building $(PROJECT_NAME)…) @
	GOOS=$(GOOS) go build -tags static -o $(PROJECT_NAME) $(BUILD_OPTS) cmd/$(PROJECT_NAME)/*.go

test: deps tidy ; $(info $(M) Testing $(PROJECT_NAME)…) @
	GOOS=$(GOOS) go test -v -cover -coverprofile=coverage.out ./...

.PHONY: lint tidy deps build test
