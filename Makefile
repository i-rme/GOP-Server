.PHONY: build test vet fmt compat compat-update

build:
	go build -o bin/gop-server ./src

test:
	go test -race ./src/...

vet:
	go vet ./src/...
	cd test/compat && go vet ./...

fmt:
	gofmt -l -w src test

# Run the compatibility suite against bin/gop-server.
compat: build
	cd test/compat && GOP_SERVER_BIN=$(CURDIR)/bin/gop-server go test -count=1 ./...

# Re-record the golden files after a deliberate behaviour change.
compat-update: build
	cd test/compat && GOP_SERVER_BIN=$(CURDIR)/bin/gop-server go test -count=1 ./... -args -update
