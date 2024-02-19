export TOOLS_PATH := ${CURDIR}/tools/bin

.PHONY: build-clean codegen codegen-clean lint lint-fix arch test tools tools-invalidate tools-clean git-pre-commit

all: lint arch test build-clean build

build: bin/duck bin/duckhandler bin/messageoutbox

build-clean:
	rm -rf bin/*

bin/%:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ./bin/$(notdir $@) ./cmd/$(notdir $@)

codegen: tools codegen-clean
	go generate ./...

codegen-clean:
	find . -type f -path "*/mock/*" -exec rm -f "{}" \;

lint: codegen tools
	tools/bin/golangci-lint --color=always run ./...

lint-fix: codegen tools
	tools/bin/golangci-lint --color=always run --fix ./...

arch: tools
	tools/bin/go-cleanarch -application app -domain domain -infrastructure infra

test: codegen
	go test ./...

tools: tools-invalidate tools/bin/mockgen tools/bin/golangci-lint tools/bin/go-cleanarch tools/bin/.go-mod.checksum

tools-invalidate:
	shasum -c ./tools/bin/.go-mod.checksum 2> /dev/null || make tools-clean

tools-clean:
	rm -rf ./tools/bin/* && rm -f ./tools/bin/.go-mod.checksum

tools/bin/mockgen:
	go build -modfile ./tools/go.mod -o ./tools/bin/mockgen go.uber.org/mock/mockgen

tools/bin/golangci-lint:
	go build -modfile ./tools/go.mod -o ./tools/bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

tools/bin/go-cleanarch:
	go build -modfile ./tools/go.mod -o ./tools/bin/go-cleanarch github.com/roblaszczak/go-cleanarch

tools/bin/.go-mod.checksum:
	shasum ./tools/go.mod ./tools/go.sum > ./tools/bin/.go-mod.checksum

git-hooks: .git/hooks/pre-commit # ignore while rebasing

git-pre-commit: lint arch test build-clean build

.git/hooks/%:
	cp  ./tools/githooks/$(notdir $@) ./.git/hooks/$(notdir $@) && \
	chmod +x ./.git/hooks/$(notdir $@)
