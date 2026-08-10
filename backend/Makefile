.PHONY: build run test vet fmt lint migrate-openapi seed clean

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

lint:
	@echo "Lint not configured yet"

migrate-openapi:
	@echo "OpenAPI validation placeholder"

seed:
	go run ./cmd/seed

clean:
	rm -rf bin/

# 全部质量门禁
check: fmt vet build test
