# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=gateway
BINARY_DIR=bin

# Build the application
build:
	$(GOBUILD) -o bin/$(BINARY_NAME) -v ./cmd/gateway

# Run the application
run:
	$(GOCMD) run ./cmd/gateway

# Generate wire code
wire:
	cd cmd/gateway && wire

# Test the application
test:
	$(GOTEST) -v ./...

# Clean build files
clean:
	$(GOCLEAN)
	rm -rf bin/

.PHONY: build run test clean wire
