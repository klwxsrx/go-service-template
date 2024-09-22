export TOOLS_BIN := ${CURDIR}/tools/bin

.PHONY: check build-clean codegen codegen-clean lint lint-fix arch test tools tools-invalidate tools-clean git-hooks-invalidate git-hooks-clean

all: check build-clean build

check: lint arch test

build: bin/duck bin/duckhandler bin/messageoutbox

build-clean:
	rm -rf bin/*

bin/%: codegen
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ./bin/$(notdir $@) ./cmd/$(notdir $@)

codegen: tools codegen-clean
	go generate ./...

codegen-clean:
	find . -type f \( -path "*/mock/*" -o -path "*/generated/*" \) -exec rm -f "{}" \;

lint: tools codegen
	tools/bin/golangci-lint --color=always run ./...

lint-fix: tools codegen
	tools/bin/golangci-lint --color=always run --fix ./...

arch: tools
	tools/bin/go-cleanarch -application app -domain domain -infrastructure infra

test: codegen
	go test ./...

tools: tools-invalidate git-hooks-invalidate \
	tools/bin/mockgen tools/bin/golangci-lint tools/bin/go-cleanarch tools/bin/goverter tools/bin/lefthook \
	tools/bin/.go-mod.checksum tools/bin/.git-hooks.checksum

tools-invalidate:
	shasum -c ./tools/bin/.go-mod.checksum > /dev/null 2>&1 || make tools-clean

tools-clean:
	rm -rf ./tools/bin/* && rm -f ./tools/bin/.go-mod.checksum

tools/bin/mockgen:
	go build -modfile ./tools/go.mod -o ./tools/bin/mockgen go.uber.org/mock/mockgen

tools/bin/golangci-lint:
	go build -modfile ./tools/go.mod -o ./tools/bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

tools/bin/go-cleanarch:
	go build -modfile ./tools/go.mod -o ./tools/bin/go-cleanarch github.com/roblaszczak/go-cleanarch

tools/bin/goverter:
	go build -modfile ./tools/go.mod -o ./tools/bin/goverter github.com/jmattheis/goverter/cmd/goverter

tools/bin/lefthook:
	go build -modfile ./tools/go.mod -o ./tools/bin/lefthook github.com/evilmartians/lefthook

git-hooks-invalidate:
	shasum -c ./tools/bin/.git-hooks.checksum ./tools/bin/.go-mod.checksum > /dev/null 2>&1 || make git-hooks-clean

git-hooks-clean:
	rm -rf .git/hooks/* && rm -f ./tools/bin/.git-hooks.checksum

tools/bin/.go-mod.checksum:
	shasum ./tools/go.mod ./tools/go.sum > ./tools/bin/.go-mod.checksum

tools/bin/.git-hooks.checksum: tools/bin/lefthook
	tools/bin/lefthook install && shasum ./.lefthook.yaml > ./tools/bin/.git-hooks.checksum
