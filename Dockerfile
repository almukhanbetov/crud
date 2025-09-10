# Этап сборки
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

# ВАЖНО: сборка статически линкованного бинарника для Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# Финальный минимальный образ
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/app .

EXPOSE 9003

CMD ["./app"]