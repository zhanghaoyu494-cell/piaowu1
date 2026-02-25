FROM golang:1.24-alpine AS builder

WORKDIR /app
# 复制整个项目以满足依赖
COPY . .
# 切换到服务目录
WORKDIR /app/gateway
RUN go mod download
RUN go build -o gateway-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gateway/gateway-service .
COPY --from=builder /app/gateway/config/config.yaml ./config/config.yaml
EXPOSE 8081
CMD ["./gateway-service"]
