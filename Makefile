BINARY  := arpdvark
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build build-all test vet update-oui clean

build:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY) .

build-all:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 .

test:
	go test -v -race -count=1 ./...

vet:
	go vet ./...

update-oui:
	bash scripts/update_oui.sh

clean:
	rm -f $(BINARY) dist/$(BINARY)-*
