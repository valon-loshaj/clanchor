BINARY := clanchor
BUILD_DIR := bin

.PHONY: build test clean

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/clanchor

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)
