FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server/main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/server /server
ENTRYPOINT ["/server"]
