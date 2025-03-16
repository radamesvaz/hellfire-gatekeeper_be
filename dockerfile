FROM golang:1.24.1 AS builder
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o app

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/app .
CMD ["./app"]
