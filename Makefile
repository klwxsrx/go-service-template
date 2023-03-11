.PHONY: clean codegen test lint arch goenv

all: clean codegen test lint arch bin/duck bin/duckhandler

clean:
	rm -rf bin/*

bin/%:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ./bin/$(notdir $@) ./cmd/$(notdir $@)

codegen:
	go generate ./...

test: codegen
	go test ./...

lint:
	golangci-lint run ./...

arch:
	go-cleanarch -application app -domain domain -infrastructure infra -interfaces integration

goenv:
	go install github.com/golang/mock/mockgen@v1.6.0
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.2
	go install github.com/roblaszczak/go-cleanarch@v1.2.1