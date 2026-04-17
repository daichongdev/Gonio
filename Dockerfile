# Stage 1: Build
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# 先拷贝依赖文件，利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/gonio cmd/server/main.go

# Stage 2: Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/gonio /app/gonio

# 配置文件需要外部挂载
# docker run -v ./config:/app/config gonio:latest

EXPOSE 8080

ENTRYPOINT ["/app/gonio"]
