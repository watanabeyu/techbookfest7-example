package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"log"
	"strconv"
)

type (
	Token struct {
		UID   uint64 `json:"uid"`
		Os    string `json:"os"`
		Token string `json:"token"`
		Arn   string `json:"arn"`
	}
)

const redisHost = "redishost"
const redisPass = "redispass"

func AddEndpoint(v string) {
	/* unmarshal */
	var t Token
	json.Unmarshal([]byte(v), &t)

	/* connect to redis */
	client := redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: redisPass,
		DB:       0,
	})

	pong, err := client.Ping().Result()
	if err != nil {
		log.Println(pong)
		panic(err)
	}

	/* save to redis */
	res := client.SAdd("user:"+strconv.FormatUint(t.UID, 10), t.Arn)
	if err := res.Err(); err != nil {
		panic(err)
	}

	fmt.Println("add arn " + t.Arn)
}

func main() {
	/* connect to redis */
	client := redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: redisPass,
		DB:       0,
	})

	pong, err := client.Ping().Result()
	if err != nil {
		log.Println(pong)
		panic(err)
	}

	/* subscribe message */
	pubsub := client.Subscribe("channel:endpoint")
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage()
		if err != nil {
			panic(err)
		}

		fmt.Println(msg.Channel)

		v := client.RPop("queue").Val()

		if v != "" {
			switch msg.Channel {
			case "channel:endpoint":
				AddEndpoint(v)
			}
		}
	}
}
