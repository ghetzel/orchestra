ORCHESTRA_CONFIG ?= config.example.yaml

default: deps test bin/orchestra

deps:
	@go get -t ./...

test:
	@go test ./...

bin:
	@mkdir ${@}
bin/orchestra: bin
	@go generate ./...
	@go build -o $(@) ./cmd/orchestra/

run: bin/orchestra
	@./bin/orchestra

.PHONY: bin/orchestra
.EXPORT_ALL_VARIABLES: