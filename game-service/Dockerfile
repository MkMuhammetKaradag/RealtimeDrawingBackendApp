# syntax=docker/dockerfile:1

FROM golang:1.23 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o auth-service ./cmd

FROM alpine:3.20
RUN adduser -D -g '' appuser && apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=builder /app/auth-service /usr/local/bin/auth-service
COPY --from=builder /app/config/config.yml /app/config/config.yml
EXPOSE 8081
USER appuser
ENV AUTH_SERVER_PORT=8081
ENTRYPOINT ["/usr/local/bin/auth-service"] 