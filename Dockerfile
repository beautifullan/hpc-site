FROM uhub.service.ucloud.cn/mirrors/golang:1.25.0 AS builder
WORKDIR /app

# 先复制依赖文件（利用缓存）
COPY go.mod go.sum ./
RUN go mod download
#copy  项目代码
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o hpc-site main.go

#运行阶段
#使用一个更小的镜像能缩小镜像体积
FROM uhub.service.ucloud.cn/openbayeshpc/alpine:3.20

# 同样使用国内源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/hpc-site .
COPY .env .
EXPOSE 8080
CMD ["./hpc-site"]