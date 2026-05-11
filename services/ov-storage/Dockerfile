FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev make

# If go.mod doesn't exist yet in real build, we init it
COPY . .
RUN if [ ! -f go.mod ]; then go mod init github.com/vnp-community/vnp-memory/services/ov-storage; go mod tidy; fi

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /go/bin/ov-storage ./cmd/server/main.go

FROM gcr.io/distroless/static-debian12:latest

COPY --from=builder /go/bin/ov-storage /bin/ov-storage

USER nonroot:nonroot

EXPOSE 9051 9104

ENTRYPOINT ["/bin/ov-storage"]
