FROM golang:1.25rc2-alpine AS builder

RUN apk add --no-cache make build-base git

COPY go.mod go.sum /src/
WORKDIR /src
RUN GOTOOLCHAIN=go1.25rc2 go mod download

COPY . /src

RUN mkdir -p ./bin && CGO_ENABLED=0 GOTOOLCHAIN=go1.25rc2 go build -ldflags "-X main.Version=demo -extldflags=-static" -o ./bin/ ./...

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata netcat-openbsd

COPY --from=builder /src/bin /app
RUN mkdir -p /app/configs
COPY --from=builder /src/configs/config.yaml /app/configs/config.yaml

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
VOLUME /data/conf

CMD ["./server", "-conf", "/app/configs"]
