.PHONY: clean lint test

all: clean lint test bin/duck

clean:
	rm -rf bin/*

bin/%:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ./bin/$(notdir $@) ./cmd/$(notdir $@)

test:
	go test ./...

lint:
	golangci-lint run