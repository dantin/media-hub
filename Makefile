# enable BASH-specific features
SHELL := /bin/bash

SOURCE_DIR := $(shell pwd)

GOFILES!=find . -name '*.go'
GOLDFLAGS := -s -w -extldflags $(LDFLAGS)

.PHONY: build
build: udp-multiplex asset-server srt-server

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
	@rm build/bin/udp-multiplex build/bin/asset-server build/bin/srt-server
	@cd third_party/srt; make clean; rm -rf CMakeCache.txt CMakeFiles Makefile cmake_install.cmake config-status.sh haisrt.pc srt.pc install_manifest.txt version.h
	@cd third_party/srt-live-server; make clean

go.sum: $(GOFILES) go.mod
	go mod tidy

asset-server udp-multiplex srt-server: $(GOFILES) go.mod go.sum
	@echo "Building" $@ "..."
	@cd cmd/$@; go build \
		-trimpath \
		-o $@ \
		-ldflags "$(GOLDFLAGS)"
	@mv cmd/$@/$@ build/bin

package:
	cd third_party/srt; ./configure; make; sudo make install
	cd third_party/srt-live-server; make
