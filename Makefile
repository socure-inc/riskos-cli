BINARY  = riskos
BUILD   = build
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "$(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)$(shell git diff --quiet || echo '-dirty')")
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build run clean cross test test-short coverage fmt lint vet pre-commit

build:
	@mkdir -p $(BUILD)
	go build $(LDFLAGS) -o $(BUILD)/$(BINARY) .

run:
	go run . $(ARGS)

clean:
	rm -rf $(BUILD)

cross:
	@mkdir -p $(BUILD)
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o $(BUILD)/riskos-darwin-arm64  .
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD)/riskos-darwin-amd64  .
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD)/riskos-linux-amd64   .
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o $(BUILD)/riskos-linux-arm64   .
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD)/riskos-windows-amd64.exe .
	@echo "Built all targets:"
	@ls -lh $(BUILD)/riskos-*

test:
	go test ./... -v

test-short:
	go test ./... -short

coverage:
	@mkdir -p $(BUILD)
	go test ./... -coverprofile=$(BUILD)/coverage.out
	go tool cover -html=$(BUILD)/coverage.out -o $(BUILD)/coverage.html
	@echo "Coverage report: $(BUILD)/coverage.html"

fmt:
	goimports -local github.com/effectivai/riskos-cli -w .

vet:
	go vet ./...

lint:
	@which golangci-lint > /dev/null 2>&1 || (echo "Install: https://golangci-lint.run/welcome/install/" && exit 1)
	golangci-lint run ./...

pre-commit: fmt vet test build
	@echo "All checks passed."
