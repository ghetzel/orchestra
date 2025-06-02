.EXPORT_ALL_VARIABLES:

DESTDIR ?= ../static

build: GOOS   := js
build: GOARCH := wasm
build:
	go get ./...
	go generate ./...
	go build -o $(DESTDIR)/main.wasm ./...