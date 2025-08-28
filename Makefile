# Makefile
BINARY_NAME=compell

build:
	@echo "Building ${BINARY_NAME}..."
	@go build -o bin/${BINARY_NAME} ./cmd/${BINARY_NAME}

run: build
	@echo "Running ${BINARY_NAME}..."
	@./bin/${BINARY_NAME}

clean:
	@echo "Cleaning..."
	@go clean
	@rm -f bin/${BINARY_NAME}

.PHONY: build run clean
