# syntax=docker/dockerfile:1

FROM golang:1.23 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gateway-service .

FROM alpine:3.20
RUN adduser -D -g '' appuser && apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=builder /app/gateway-service /usr/local/bin/gateway-service
EXPOSE 8080
USER appuser
ENTRYPOINT ["/usr/local/bin/gateway-service"] 