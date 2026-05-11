.PHONY: build test lint mock generate run clean docker-build docker-up

BINARY_NAME=ov-storage
MAIN_PATH=cmd/server/main.go

build:
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

run: build
	./bin/$(BINARY_NAME)

test:
	go test -v -cover -race ./...

lint:
	golangci-lint run

mock:
	mockgen -source=internal/usecase/port/interfaces.go -destination=internal/usecase/port/mock/interfaces_mock.go -package=mock

generate:
	go generate ./...

clean:
	go clean
	rm -rf bin/

docker-build:
	docker build -t vnp-community/$(BINARY_NAME):latest .

docker-up:
	docker-compose up -d
