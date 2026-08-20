.PHONY: build test vet clean run cross install

BINARY=pollkit
VERSION=$(shell cat VERSION 2>/dev/null || echo "0.1.0")

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/pollkit

test:
	go test ./...

vet:
	go vet ./...

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY) $(BINARY)-*

install: build
	cp $(BINARY) /usr/local/bin/

cross:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -o $(BINARY)-linux-amd64   ./cmd/pollkit
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -trimpath -o $(BINARY)-linux-arm64   ./cmd/pollkit
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -trimpath -o $(BINARY)-darwin-amd64  ./cmd/pollkit
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -o $(BINARY)-darwin-arm64  ./cmd/pollkit
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o $(BINARY)-windows-amd64 ./cmd/pollkit
