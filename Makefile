.PHONY: build vet compat compat-update

build:
	go build -o bin/gop-server ./src

vet:
	go vet ./src/...
	cd test/compat && go vet ./...

# Run the compatibility suite against bin/gop-server.
# On Linux the current server must run as root (see test/compat/README.md).
compat: build
	cd test/compat && GOP_SERVER_BIN=$(CURDIR)/bin/gop-server go test -count=1 ./...

# Re-record the golden files after a deliberate behaviour change.
compat-update: build
	cd test/compat && GOP_SERVER_BIN=$(CURDIR)/bin/gop-server go test -count=1 ./... -args -update
