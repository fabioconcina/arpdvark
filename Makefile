BINARY  := arpdvark
VERSION ?= dev
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build build-all update-oui clean

build:
	go build $(LDFLAGS) -o $(BINARY) .

build-all:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 .

update-oui:
	bash scripts/update_oui.sh

clean:
	rm -f $(BINARY) dist/$(BINARY)-*
