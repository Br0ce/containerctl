.PHONY: build format lint clean-test test test-v test-race clean run tidy setup demo-up demo-down
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
	rm -f ./bin/*

run:
	go run ./cmd $(ARGS)

build:
	go build -o bin/cctl ./cmd

tidy:
	go mod tidy
	go mod vendor

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
	go mod tidy

demo-up:
	docker compose -f docs/demo/compose.yaml up -d
	docker compose -f docs/demo/compose.yaml pause worker
	docker compose -f docs/demo/monitoring.yaml up -d
	docker compose -f docs/demo/monitoring.yaml --profile manual create reporter
	docker rm -f scratch >/dev/null 2>&1; docker run -d --name scratch busybox sleep infinity

demo-down:
	docker compose -f docs/demo/compose.yaml down
	docker compose -f docs/demo/monitoring.yaml --profile manual down
	docker rm -f scratch
