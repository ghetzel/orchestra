ORCHESTRA_CONFIG ?= config.example.yaml

default: deps test build

deps:
	@which vgrun > /dev/null 2>&1 || go install github.com/vugu/vgrun@latest
	@which vugugen vgrgen > /dev/null 2>&1 || vgrun -install-tools
	@go get -t ./...

test:
	@go test ./...

build: static/main.wasm bin/orchestra

bin:
	@mkdir ${@}
bin/orchestra: bin
	@go generate .
	@go build -o $(@) ./cmd/orchestra/

static/main.wasm:
	@make -C ui -B

run: bin/orchestra
	@./bin/orchestra

.PHONY: static/main.wasm bin/orchestra 
.EXPORT_ALL_VARIABLES: