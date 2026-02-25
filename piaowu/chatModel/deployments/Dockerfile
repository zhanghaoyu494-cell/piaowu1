FROM golang:1.24-alpine AS builder

WORKDIR /app
# 复制整个项目以满足依赖 (monorepo)
COPY . .

# 下载依赖并构建
RUN go mod download
RUN go build -o chat-model chatModel/cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/chat-model .
# 如果有配置文件，也需要复制
# COPY --from=builder /app/chatModel/config ./config

EXPOSE 8082
CMD ["./chat-model"]
