export TOOLS_PATH := ${CURDIR}/tools/bin

.PHONY: clean codegen lint lint-fix arch test tools-clean

all: clean lint arch test build

clean:
	rm -rf bin/*

build: bin/duck bin/duckhandler bin/messageoutbox

bin/%:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ./bin/$(notdir $@) ./cmd/$(notdir $@)

codegen: tools/bin/mockgen
	go generate ./...

lint: codegen tools/bin/golangci-lint
	tools/bin/golangci-lint run ./...

lint-fix: codegen tools/bin/golangci-lint
	tools/bin/golangci-lint run --fix ./...

arch: tools/bin/go-cleanarch
	tools/bin/go-cleanarch -interfaces api -application app -domain domain -infrastructure infra

test: codegen
	go test ./...

tools: tools/bin/mockgen tools/bin/golangci-lint tools/bin/go-cleanarch

tools/bin/mockgen:
	go build -modfile ./tools/go.mod -o ./tools/bin/mockgen go.uber.org/mock/mockgen

tools/bin/golangci-lint:
	go build -modfile ./tools/go.mod -o ./tools/bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

tools/bin/go-cleanarch:
	go build -modfile ./tools/go.mod -o ./tools/bin/go-cleanarch github.com/roblaszczak/go-cleanarch

tools-clean:
	rm -rf tools/bin/*