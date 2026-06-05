BINARY  = riskos
VERSION = 1.0.0
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build run clean cross test test-short coverage fmt lint

build:
	go build $(LDFLAGS) -o $(BINARY) .

run:
	go run . $(ARGS)

clean:
	rm -f $(BINARY) riskos-*

cross:
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o riskos-darwin-arm64  .
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o riskos-darwin-amd64  .
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o riskos-linux-amd64   .
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o riskos-linux-arm64   .
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o riskos-windows-amd64.exe .
	@echo "Built all targets:"
	@ls -lh riskos-*

test:
	go test ./... -v

test-short:
	go test ./... -short

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

fmt:
	go fmt ./...

lint:
	go vet ./...
