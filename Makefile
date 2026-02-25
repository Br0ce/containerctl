.PHONY: build format test clean test-e2e test-e2e-verbose test-e2e-parallel test-e2e-html

format:
	go fmt ./...

lint:
	golangci-lint run ./...

clean-test:
	go clean -testcache

test:
	$(MAKE) clean-test && go test -parallel 4 ./... -short

test-v:
	$(MAKE) clean-test && go test -v -cover ./... -short

test-race:
	$(MAKE) clean-test && go test -race ./... -short

clean:
	rm -f ./bin/

run:
	go run ./cmd

build:
	go build -o bin/cctl ./cmd

tidy:
	go mod tidy
	go mod vendor

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1
	go mod tidy