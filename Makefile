BINARY_NAME := openapi-go-md
BIN_DIR := bin

.PHONY: all build test lint fmt tidy clean

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/openapi-go-md

test:
	go test ./...

lint:
	go vet ./...

fmt:
	gofmt -w ./cmd ./pkg

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)
