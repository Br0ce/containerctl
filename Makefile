.PHONY: build format lint clean-test test test-v test-race clean run build tidy setup
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
	go run ./cmd $(ARGS)

build:
	go build -o bin/containerctl ./cmd

tidy:
	go mod tidy
	go mod vendor

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1
	go mod tidy