.PHONY: test test-integration build run

build:
	go build -o bin/server ./cmd/server/main.go

run: build
	./bin/server

test:
	go test -v -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}' | awk '{if ($$1 < 80) {print "Coverage is below 80%"; exit 1} else {print "Coverage is " $$1 "%. Passed."}}'

test-integration:
	docker-compose -f docker-compose.test.yml up -d
	sleep 5 # wait for services
	go test -v -tags=integration ./tests/...
	docker-compose -f docker-compose.test.yml down
