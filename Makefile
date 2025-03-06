.PHONY: build install clean test

# Variables
BINARY_NAME=supper
VERSION=0.1.0
BUILD_DIR=build
INSTALL_PATH=$(HOME)/.local/bin

# Go build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/supper

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@mkdir -p $(INSTALL_PATH)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installation complete. Make sure $(INSTALL_PATH) is in your PATH."

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

# Run tests
test:
	go test -v ./...

# Run the binary
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

# Format code
fmt:
	go fmt ./...

# Check for common mistakes and errors
lint:
	go vet ./...
	@if command -v golint >/dev/null 2>&1; then \
		golint ./...; \
	else \
		echo "golint not installed. Run: go get -u golang.org/x/lint/golint"; \
	fi

# Dependencies
deps:
	@echo "Ensuring dependencies..."
	go mod tidy
	go mod download

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the binary"
	@echo "  install  - Install the binary to $(INSTALL_PATH)"
	@echo "  clean    - Remove build artifacts"
	@echo "  test     - Run tests"
	@echo "  run      - Build and run the binary"
	@echo "  fmt      - Format code"
	@echo "  lint     - Check for common mistakes and errors"
	@echo "  deps     - Ensure dependencies are installed"
	@echo "  help     - Show this help message"

# Version information
version:
	@echo "$(BINARY_NAME) $(VERSION)"
