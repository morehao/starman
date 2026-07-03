.PHONY: all
all: build

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o starman ./cmd/starman/

.PHONY: build-release
build-release:
	CGO_ENABLED=0 go build -ldflags="-s -w -X github.com/morehao/starman/internal/version.ver=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)" -trimpath -o starman ./cmd/starman/

.PHONY: install
install:
	go install -ldflags="-s -w" -trimpath ./cmd/starman/

.PHONY: run
run: build
	./starman

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	golangci-lint run 2>/dev/null || echo "golangci-lint not installed"

.PHONY: clean
clean:
	rm -f starman
