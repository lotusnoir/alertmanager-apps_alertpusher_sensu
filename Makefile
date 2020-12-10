REVISION=$(shell git describe --tags --always)
BUILD=$(shell date +%FT%T%z)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
LDFLAGS=-ldflags "-X main.Revision=${REVISION} -X main.Build=${BUILD} -X main.Branch=${BRANCH}"

all:
	go build $(LDFLAGS)

install: # installs to $GOPATH/bin
	go install $(LDFLAGS)

clean:
	go clean -i -testcache -modcache

.PHONY: all install  clean
