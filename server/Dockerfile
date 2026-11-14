# 构建阶段：纯 Go 依赖，CGO_ENABLED=0 产出静态二进制。
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
        -o /out/kindergarten-server ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
        -o /out/kindergarten-migrate ./cmd/migrate && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
        -o /out/kindergarten-seedbase ./cmd/seedbase

# 运行阶段：只带二进制、Casbin 策略与 CA/时区数据。
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
ENV TZ=Asia/Shanghai
ENV CONFIG_PATH=/etc/kindergarten/app.ini
WORKDIR /app
COPY --from=build /out/kindergarten-server /app/kindergarten-server
COPY --from=build /out/kindergarten-migrate /app/kindergarten-migrate
COPY --from=build /out/kindergarten-seedbase /app/kindergarten-seedbase
COPY configs/rbac_model.conf configs/rbac_policy.csv /app/configs/

EXPOSE 8080
# 存活探针不依赖 MySQL/Redis；就绪探针由部署流水线访问 /ready。
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/kindergarten-server"]
