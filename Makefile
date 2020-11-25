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
clean: clean_srt clean_sls
	@echo "Running clean..."
	@go clean
	@rm -f build/bin/udp-multiplex build/bin/asset-server build/bin/srt-server
	@rm -rf build/tmp
	@rm -f config/*.pid config/*.conf

go.sum: $(GOFILES) go.mod
	go mod tidy

asset-server udp-multiplex srt-server: $(GOFILES) go.mod go.sum
	@echo "Building" $@ "..."
	@cd cmd/$@; go build \
		-trimpath \
		-o $@ \
		-ldflags "$(GOLDFLAGS)"
	@mv cmd/$@/$@ build/bin

.PHONY: build_srt
build_srt:
	@cd third_party/srt; ./configure; make; sudo make install

.PHONY: clean_srt
clean_srt:
	@cd third_party/srt; make clean; rm -rf CMakeCache.txt CMakeFiles Makefile cmake_install.cmake config-status.sh haisrt.pc srt.pc install_manifest.txt version.h

.PHONY: build_sls
build_sls: build_srt
	@cd third_party/srt-live-server; make

.PHONY: clean_sls
clean_sls:
	@cd third_party/srt-live-server; make clean

package: build_sls build
	@mkdir -p build/tmp/ultrasound_1.0-1
	@mkdir -p build/tmp/ultrasound_1.0-1/usr/local/bin
	@mkdir -p build/tmp/ultrasound_1.0-1/usr/local/include/srt
	@mkdir -p build/tmp/ultrasound_1.0-1/usr/local/lib/pkgconfig
	@cp -r build/DEBIAN build/tmp/ultrasound_1.0-1
	@cp build/bin/srt-server build/tmp/ultrasound_1.0-1/usr/local/bin
	@cp third_party/srt/srt-live-transmit build/tmp/ultrasound_1.0-1/usr/local/bin
	@cp third_party/srt-live-server/bin/slc build/tmp/ultrasound_1.0-1/usr/local/bin
	@cp third_party/srt-live-server/bin/sls build/tmp/ultrasound_1.0-1/usr/local/bin
	@cp third_party/srt/*.pc build/tmp/ultrasound_1.0-1/usr/local/lib/pkgconfig
	@cp third_party/srt/libsrt.a build/tmp/ultrasound_1.0-1/usr/local/lib
	@cp third_party/srt/libsrt.so* build/tmp/ultrasound_1.0-1/usr/local/lib
	@cd build/tmp; dpkg-deb --build ultrasound_1.0-1
