GOPATH := $(shell pwd)

all: build

bindata: $(shell find assets -type f)
	@go get github.com/jteeuwen/go-bindata/...
	@$(GOPATH)/bin/go-bindata -pkg=main -o=assets.go assets

build: $(shell find . -name '*.go') bindata
	@go get github.com/mtojo/go-java/java
	@mkdir -p bin
	@go build -o bin/j2n *.go

lint:
	@go get github.com/golang/lint/golint
	@$(GOPATH)/bin/golint *.go

.PHONY: all bindata build lint
