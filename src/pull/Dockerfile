FROM registry.cn-huhehaote.aliyuncs.com/feng-566/golang:1.14-alpine AS builder
ENV GO111MODULE=on \
	GOPROXY=https://goproxy.cn
WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build -o main ./

FROM registry.cn-huhehaote.aliyuncs.com/feng-566/alpine:3.11
COPY --from=builder /go/src/app/main /main
RUN chmod +x /main

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
ENTRYPOINT ["/main"]
