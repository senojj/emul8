GO_CMD = go
GO_BUILD = $(GO_CMD) build
GO_TEST = $(GO_CMD) test
GO_CLEAN = $(GO_CMD) clean
BINARY_NAME = emul8
BUILD_DIR = ./bin
MAIN_SRC = ./cmd/emul8/main.go

.PHONY: all build test clean run

all: build

build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME): $(MAIN_SRC)
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) -o $@ $<

test:
	$(GO_TEST) ./...

clean:
	$(GO_CLEAN)
	@rm -rf $(BUILD_DIR)

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

help:
	@echo "Available commands:"
	@echo "  make build    Compile the binary."
	@echo "  make test     Run all tests."
	@echo "  make clean    Remove build files and the binary."
	@echo "  make run      Build and run the application."
	@echo "  make all      Default target, runs build."
