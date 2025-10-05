.PHONY: check build-clean codegen codegen-clean lint arch test git-hooks

all: check build-clean build

check: lint arch test

build: bin/user-service bin/user-profile-service bin/message-handler-worker bin/idk-cleaner-task

bin/%: codegen
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ./bin/$(notdir $@) ./cmd/$(notdir $@)

build-clean:
	rm -rf bin/*

codegen: codegen-clean
	go generate ./...

codegen-clean:
	find . -type f \( -path "*/generated/*" \) -exec rm -f "{}" \;

lint: codegen
	go tool golangci-lint --color=always run ./...

arch:
	go tool go-cleanarch -interfaces api -application app -domain domain -infrastructure infra

test: codegen
	go test ./...

git-hooks:
	go tool lefthook check-install || go tool lefthook install
