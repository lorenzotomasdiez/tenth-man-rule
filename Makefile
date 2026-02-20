.PHONY: build test lint fmt clean run

BINARY=tenthman

build:
	go build -o $(BINARY) ./cmd/tenthman

test:
	go test -race ./...

test-verbose:
	go test -v -race ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

clean:
	rm -f $(BINARY)
	rm -rf output/

run: build
	./$(BINARY) debate --topic "$(TOPIC)"
