package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
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

	Action struct {
		To     uint64 `json:"to"`
		From   uint64 `json:"from"`
		Action string `json:"action"`
	}

	PublishData struct {
		Alert *string     `json:"alert,omitempty"`
		Sound *string     `json:"sound,omitempty"`
		Data  interface{} `json:"custom_data"`
		Badge *int        `json:"badge,omitempty"`
	}

	iosPush struct {
		APS PublishData `json:"aps"`
	}

	gcmPush struct {
		Message *string `json:"message,omitempty"`
	}

	gcmPushWrapper struct {
		Data gcmPush `json:"data"`
	}

	publishMessage struct {
		APNS        string `json:"APNS"`
		APNSSandbox string `json:"APNS_SANDBOX"`
		Default     string `json:"default"`
		GCM         string `json:"GCM"`
	}
)

const awsAccessKey = "awsAccessKey"
const awsAccessSecret = "awsAccessSecret"
const awsRegion = "awsRegion"
const redisHost = "redishost"
const redisPass = "redispass"

func NewAwsSession(conf *aws.Config) (*session.Session, error) {
	region := awsRegion
	conf.Credentials = credentials.NewStaticCredentials(
		awsAccessKey,
		awsAccessSecret,
		"",
	)
	conf.Region = &region

	sess, err := session.NewSession(conf)

	if err != nil {
		log.Println("create aws session failed")

		return nil, err
	}

	return session.Must(sess, err), nil
}

func SnsPublish(endpoints []string, data *PublishData) {
	/* json */
	b, err := json.Marshal(iosPush{
		APS: *data,
	})

	if err != nil {
		panic(err)
	}

	payload := string(b)

	/* gcm */
	b, err = json.Marshal(gcmPushWrapper{
		Data: gcmPush{
			Message: data.Alert,
		},
	})

	if err != nil {
		panic(err)
	}

	gcm := string(b)

	/* message */
	pushData, err := json.Marshal(publishMessage{
		Default:     *data.Alert,
		APNS:        payload,
		APNSSandbox: payload,
		GCM:         gcm,
	})

	if err != nil {
		panic(err)
	}

	/* aws session */
	sess, err := NewAwsSession(&aws.Config{})
	if err != nil {
		panic(err)
	}

	/* sns session */
	svc := sns.New(sess)

	for _, v := range endpoints {
		_, err = svc.Publish(&sns.PublishInput{
			Message:          aws.String(string(pushData)),
			MessageStructure: aws.String("json"),
			TargetArn:        aws.String(v),
		})

		if err != nil {
			log.Println(err)
		}
	}
}

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

func pushToUser(v string) {
	/* unmarshal */
	var p Action
	json.Unmarshal([]byte(v), &p)

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

	/* get endpoint arn */
	key := "user:" + strconv.FormatUint(p.To, 10) + ":endpoint"
	endpoints := client.SMembers(key).Val()

	SnsPublish(endpoints, &PublishData{
		Alert: aws.String("ユーザID" + strconv.FormatUint(p.From, 10) + "さんにいいねされました"),
		Sound: aws.String("default"),
		Badge: aws.Int(1),
	})
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
	pubsub := client.Subscribe("channel:endpoint", "channel:like")
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
				AddEndpoint(msg.Payload)
			case "channel:like":
				pushToUser(msg.Payload)
			}
		}
	}
}
