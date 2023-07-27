export PROJECT_PATH := $(CURDIR)

.PHONY: clean codegen test lint arch tools

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
	tools/golangci-lint run ./...

arch: tools
	tools/go-cleanarch -application app -domain domain -infrastructure infra -interfaces integration

tools: tools/mockgen tools/golangci-lint tools/go-cleanarch

tools/mockgen:
	go build -modfile ./.tools/go.mod -o ./tools/mockgen go.uber.org/mock/mockgen

tools/golangci-lint:
	go build -modfile ./.tools/go.mod -o ./tools/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

tools/go-cleanarch:
	go build -modfile ./.tools/go.mod -o ./tools/go-cleanarch github.com/roblaszczak/go-cleanarch