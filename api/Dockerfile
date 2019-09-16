FROM golang:1.13

WORKDIR /go/src/app

ENV GO111MODULE=on

RUN go get -u github.com/go-redis/redis
RUN go get -u github.com/labstack/echo/...
RUN go get -u github.com/aws/aws-sdk-go
RUN go get -u github.com/aws/aws-sdk-go/aws/...
RUN go get -u github.com/aws/aws-sdk-go/service/...

EXPOSE 8080

CMD ["go", "run", "main.go"]