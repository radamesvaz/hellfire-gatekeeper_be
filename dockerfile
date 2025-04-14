FROM golang:1.24.1 AS builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o app ./cmd/api

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/app .

CMD ["./app"]
