export TOOLS_PATH := ${CURDIR}/tools/bin

.PHONY: clean codegen lint lint-fix arch test tools tools-invalidate tools-clean git-pre-commit

all: lint arch test clean build

clean:
	rm -rf bin/*

build: bin/duck bin/duckhandler bin/messageoutbox

bin/%:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ./bin/$(notdir $@) ./cmd/$(notdir $@)

codegen: tools
	go generate ./...

lint: codegen tools
	tools/bin/golangci-lint --color=always run ./...

lint-fix: codegen tools
	tools/bin/golangci-lint --color=always run --fix ./...

arch: tools
	tools/bin/go-cleanarch -interfaces api -application app -domain domain -infrastructure infra

test: codegen
	go test ./...

tools: tools-invalidate tools/bin/mockgen tools/bin/golangci-lint tools/bin/go-cleanarch tools/bin/.go-mod.checksum

tools-invalidate:
	shasum -c ./tools/bin/.go-mod.checksum 2> /dev/null || make tools-clean

tools-clean:
	rm -rf ./tools/bin/*

tools/bin/mockgen:
	go build -modfile ./tools/go.mod -o ./tools/bin/mockgen go.uber.org/mock/mockgen

tools/bin/golangci-lint:
	go build -modfile ./tools/go.mod -o ./tools/bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

tools/bin/go-cleanarch:
	go build -modfile ./tools/go.mod -o ./tools/bin/go-cleanarch github.com/roblaszczak/go-cleanarch

tools/bin/.go-mod.checksum:
	shasum ./tools/go.mod ./tools/go.sum > ./tools/bin/.go-mod.checksum

git-hooks: .git/hooks/pre-commit

git-pre-commit: lint arch test clean build

.git/hooks/%:
	cp  ./tools/githooks/$(notdir $@) ./.git/hooks/$(notdir $@) && \
	chmod +x ./.git/hooks/$(notdir $@)
