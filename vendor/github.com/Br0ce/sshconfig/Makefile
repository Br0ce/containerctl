.PHONY: format lint clean-test test test-v test-race tidy setup
format:
	go fmt ./...

lint:
	golangci-lint run ./...

clean-test:
	go clean -testcache

test:
	$(MAKE) clean-test && go test ./...

test-v:
	$(MAKE) clean-test && go test -v -cover ./...

test-race:
	$(MAKE) clean-test && go test -race ./...

tidy:
	go mod tidy

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
