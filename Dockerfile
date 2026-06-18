FROM golang:1.25.0-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/kg-service .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/kg-service /usr/local/bin/kg-service

EXPOSE 8082

ENTRYPOINT ["/usr/local/bin/kg-service"]
