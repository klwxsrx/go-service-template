export TOOLS_PATH := ${CURDIR}/tools/bin

.PHONY: clean codegen test lint arch tools-clean tools-update tools

all: clean codegen test lint arch bin/duck bin/duckhandler

clean:
	rm -rf bin/*

bin/%:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ./bin/$(notdir $@) ./cmd/$(notdir $@)

codegen: tools
	go generate ./...

test: codegen
	go test ./...

lint: tools
	tools/bin/golangci-lint run ./...

lint-fix: tools
	tools/bin/golangci-lint run --fix ./...

arch: tools
	tools/bin/go-cleanarch -application app -domain domain -infrastructure infra -interfaces integration

tools: tools/bin/mockgen tools/bin/golangci-lint tools/bin/go-cleanarch

tools/bin/mockgen:
	go build -modfile ./tools/go.mod -o ./tools/bin/mockgen go.uber.org/mock/mockgen

tools/bin/golangci-lint:
	go build -modfile ./tools/go.mod -o ./tools/bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

tools/bin/go-cleanarch:
	go build -modfile ./tools/go.mod -o ./tools/bin/go-cleanarch github.com/roblaszczak/go-cleanarch

tools-clean:
	rm -rf tools/bin/*

tools-update: tools-clean tools