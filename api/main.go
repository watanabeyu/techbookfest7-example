package main

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/go-redis/redis"
	"github.com/labstack/echo"
	"log"
	"net/http"
	"strconv"
)

type (
	Token struct {
		UID     uint64 `json:"uid"`
		Os      string `json:"os"`
		Token   string `json:"token"`
		Arn     string `json:"arn"`
		Channel string `json:"channel"`
	}

	SnsEndPointCreateResult struct {
		EndpointArn *string
	}

	Action struct {
		To      uint64 `json:"to"`
		From    uint64 `json:"from"`
		Action  string `json:"action"`
		Channel string `json:"channel"`
	}

	ActionResponse struct {
		Action string `json:"action"`
		Error  bool   `json:"error"`
	}
)

const awsAccessKey = "awsAccessKey"
const awsAccessSecret = "awsAccessSecret"
const awsRegion = "awsRegion"
const iosArn = "iosArn"
const androidArn = "androidArn"
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

func main() {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.POST("/token", func(c echo.Context) error {
		/* get json */
		p := new(Token)
		if err := c.Bind(p); err != nil {
			return err
		}

		/* aws session */
		sess, err := NewAwsSession(&aws.Config{})
		if err != nil {
			panic(err)
		}

		svc := sns.New(sess)

		/* create endpoint */
		var arn string
		if p.Os == "ios" {
			arn = iosArn
		} else {
			arn = androidArn
		}

		userData := "user_" + strconv.FormatUint(p.UID, 10)
		res, err := svc.CreatePlatformEndpoint(&sns.CreatePlatformEndpointInput{
			CustomUserData:         aws.String(userData),
			PlatformApplicationArn: aws.String(arn),
			Token:                  aws.String(p.Token),
		})

		if err != nil {
			panic(err)
		}

		p.Arn = *res.EndpointArn

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

		/* set queue */
		q, err := json.Marshal(p)
		if err != nil {
			panic(err)
		}

		client.LPush("queue", string(q))
		client.Publish("channel:endpoint", string(q))
		client.Close()

		/* return response */
		return c.JSON(http.StatusOK, p)
	})

	e.POST("/action", func(c echo.Context) error {
		/* get json */
		p := new(Action)
		if err := c.Bind(p); err != nil {
			return err
		}

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

		/* execute action */
		var r ActionResponse
		r.Action = p.Action
		r.Error = false

		/* set queue */
		q, err := json.Marshal(p)
		if err != nil {
			panic(err)
		}

		client.LPush("queue", string(q))
		client.Publish("channel:"+p.Action, string(q))
		client.Close()

		/* return response */
		return c.JSON(http.StatusOK, r)
	})

	e.Logger.Fatal(e.Start(":8080"))
}
