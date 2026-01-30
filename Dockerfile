# Dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/mclog-parser ./cmd/main.go

# Финальный образ
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/mclog-parser .

# Порт не нужен (CLI-приложение), но оставим на будущее
EXPOSE 8080

CMD ["./mclog-parser"]