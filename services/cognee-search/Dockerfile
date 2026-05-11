FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
# COPY go.sum ./
# RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /go/bin/server cmd/server/main.go

FROM scratch
COPY --from=builder /go/bin/server /server
ENTRYPOINT ["/server"]
