FROM golang:1.25-alpine AS builder

WORKDIR /app

# Установка зависимостей для сборки
RUN apk add --no-cache git

# Копирование go mod файлов
COPY go.mod go.sum* ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Сборка индексера
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o indexer ./cmd/indexer

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Копирование бинарников из builder
COPY --from=builder /app/main .
COPY --from=builder /app/indexer .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/front ./front
COPY --from=builder /app/js_API_Ya_map ./js_API_Ya_map

EXPOSE 8080

CMD ["./main"]
