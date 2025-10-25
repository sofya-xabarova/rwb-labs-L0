# syntax=docker/dockerfile:1
FROM golang:1.24.9 AS builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o ordersvc

# ------------------------
FROM debian:bookworm-slim
WORKDIR /app

COPY --from=builder /app/ordersvc .
COPY static ./static
COPY schema.sql ./schema.sql

EXPOSE 8080
# Заменяем CMD на ENTRYPOINT и оставляем CMD пустым
ENTRYPOINT ["./ordersvc"]
CMD ["--help"]  # или просто CMD []