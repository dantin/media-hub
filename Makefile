# enable BASH-specific features
SHELL := /bin/bash

SOURCE_DIR := $(shell pwd)

GOFILES!=find . -name '*.go'
GOLDFLAGS := -s -w -extldflags $(LDFLAGS)

.PHONY: build
build: udp-multiplex asset-server

.PHONY: test
test:
	@echo "Running tests..."
	@go test -race $$(go list ./...)

.PHONY: coverage
coverage:
	@echo "Running coverage..."
	@go test -race -coverprofile=coverage.txt -covermode=atomic $$(go list ./...)

.PHONY: vet
vet:
	@echo "Running vet..."
	@go vet $$(go list ./...)

.PHONY: lint
lint:
	@echo "Running lint..."
	@golint $$(go list ./...)

.PHONY: clean
clean:
	@echo "Running clean..."
	@go clean
	@rm bin/udp-multiplex bin/asset-server

go.sum: $(GOFILES) go.mod
	go mod tidy

asset-server udp-multiplex: $(GOFILES) go.mod go.sum
	@echo "Building" $@ "..."
	@cd cmd/$@; go build \
		-trimpath \
		-o $@ \
		-ldflags "$(GOLDFLAGS)"
	@mv cmd/$@/$@ bin
