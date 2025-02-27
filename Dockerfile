FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o service ./cmd/service

FROM alpine:latest
COPY --from=builder /app/service /service
COPY .env .env
EXPOSE 8080
ENTRYPOINT ["/service"]