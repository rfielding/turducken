.PHONY: build run clean test deps

BINARY=turducken
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build: deps
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/turducken

run: build
	./$(BINARY)

deps:
	go mod download

test:
	go test -v ./...

clean:
	rm -f $(BINARY)
	go clean

# Release builds for multiple platforms
release: deps
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)-linux-amd64 ./cmd/turducken
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)-darwin-amd64 ./cmd/turducken
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BINARY)-darwin-arm64 ./cmd/turducken
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)-windows-amd64.exe ./cmd/turducken

# Run with example spec
demo: build
	./$(BINARY) -spec specs/two_phase_commit.pl
