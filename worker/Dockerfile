FROM golang:1.13

COPY . /go/src/app
WORKDIR /go/src/app

ENV GO111MODULE=on

RUN go get -u github.com/go-redis/redis

EXPOSE 8080

CMD ["go", "run", "main.go"]