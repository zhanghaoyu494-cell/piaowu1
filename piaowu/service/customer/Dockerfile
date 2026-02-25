FROM golang:1.24-alpine AS builder

WORKDIR /app
# 复制整个项目以满足依赖
COPY . .
# 切换到服务目录
WORKDIR /app/service/customer
RUN go mod download
RUN go build -o customer-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/service/customer/customer-service .
COPY --from=builder /app/service/customer/config/config.yaml ./config/config.yaml
EXPOSE 8888
CMD ["./customer-service"]
