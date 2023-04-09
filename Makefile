BINARY_NAME=vim-chatgpt
SHELL=/bin/bash

.PHONY: all
## : Same as 'make download build', recommended after checking out
all: download build

.PHONY: help
## help: Prints this help
help:
	@sed -ne 's/^##/make/p' $(MAKEFILE_LIST) | column -c2 -t -s ':' | sort

.PHONY: build
## build: Builds binary for current OS
build:
	go build -o $(BINARY_NAME) -v ./*.go

.PHONY: clean
## clean: Clean up build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

.PHONY: download
## download: Download dependencies
download:
	go mod download

.PHONY: fmt
## fmt: Run 'go fmt' on all source files
fmt:
	go fmt ./...

.PHONY: vet
## vet: Run 'go vet' on all source files
vet:
	go vet ./...
