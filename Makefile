export TOOLS_PATH := ${CURDIR}/tools/bin

.PHONY: dev-tools clean codegen lint lint-fix arch test tools tools-update tools-clean git-pre-commit git-post-checkout

all: lint arch test clean build

dev-tools: tools git-hooks

clean:
	rm -rf bin/*

build: bin/duck bin/duckhandler bin/messageoutbox

bin/%:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ./bin/$(notdir $@) ./cmd/$(notdir $@)

codegen: tools/bin/mockgen
	go generate ./...

lint: codegen tools/bin/golangci-lint
	tools/bin/golangci-lint --color=always run ./...

lint-fix: codegen tools/bin/golangci-lint
	tools/bin/golangci-lint --color=always run --fix ./...

arch: tools/bin/go-cleanarch
	tools/bin/go-cleanarch -interfaces api -application app -domain domain -infrastructure infra

test: codegen
	go test ./...

tools: tools/bin/mockgen tools/bin/golangci-lint tools/bin/go-cleanarch
	shasum ./tools/go.mod ./tools/go.sum > ./tools/bin/.go-mod.checksum

tools/bin/mockgen:
	go build -modfile ./tools/go.mod -o ./tools/bin/mockgen go.uber.org/mock/mockgen

tools/bin/golangci-lint:
	go build -modfile ./tools/go.mod -o ./tools/bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

tools/bin/go-cleanarch:
	go build -modfile ./tools/go.mod -o ./tools/bin/go-cleanarch github.com/roblaszczak/go-cleanarch

tools-update:
	shasum -c ./tools/bin/.go-mod.checksum 2>/dev/null || make tools-clean tools

tools-clean:
	rm -rf ./tools/bin/*

git-hooks: .git/hooks/pre-commit .git/hooks/post-checkout

.git/hooks/%:
	cp  ./tools/githooks/$(notdir $@) ./.git/hooks/$(notdir $@) && \
	chmod +x ./.git/hooks/$(notdir $@)

git-pre-commit: lint arch test clean build

git-post-checkout: tools-update
