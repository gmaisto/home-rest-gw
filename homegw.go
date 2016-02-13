package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"
)

var wmux sync.Mutex
var datamux sync.Mutex
var fgdata string
var lastseen string
var light1 string
var light1ls string

type hwdata struct {
	Message  string `json:"message"`
	Lastseen string `json:"lastseen"`
	Lux      string `json:"lux"`
	Temp     string `json:"temp"`
	Lstate   string `json:"lstate"`
	Light1   string `json:"light1"`
	Light1ls string `json:"light1ls"`
}

var curdata hwdata

func mqttWorker(c *client.Client) {

	// Subscribe to topics.
	err := c.Subscribe(&client.SubscribeOptions{
		SubReqs: []*client.SubReq{
			&client.SubReq{
				TopicFilter: []byte("fgdata"),
				QoS:         mqtt.QoS0,
				// Define the processing of the message handler.
				Handler: func(topicName, message []byte) {
					wmux.Lock()
					fgdata = string(message)
					lastseen = time.Now().String()
					wmux.Unlock()
				},
			},

			&client.SubReq{
				TopicFilter: []byte("light1"),
				QoS:         mqtt.QoS0,
				// Define the processing of the message handler.
				Handler: func(topicName, message []byte) {
					wmux.Lock()
					light1 = string(message)
					light1ls = time.Now().String()
					wmux.Unlock()
				},
			},
		},
	})
	if err != nil {
		fmt.Println("Error", err)
	}
}

func main() {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":  "pong",
			"lastseen": time.Now().String(),
		})
	})

	r.GET("/fgdata", func(c *gin.Context) {

		datamux.Lock()
		if fgdata != "" && strings.Contains(fgdata, "#") {
			s := strings.Split(fgdata, "#")

			curdata.Message = "OK"
			curdata.Lastseen = lastseen
			curdata.Lux = s[1]
			curdata.Temp = s[2]
			curdata.Lstate = s[0]
		}

		if light1 != "" {
			curdata.Light1 = light1
			curdata.Light1ls = light1ls
		} else {
			curdata.Light1 = "N/A"
			curdata.Light1ls = light1ls
		}
		datamux.Unlock()

		if fgdata != "" && strings.Contains(fgdata, "#") {
			//s := strings.Split(fgdata, "#")
			c.JSON(200, curdata)
			// c.JSON(200, gin.H{
			// 	"message":  "OK",
			// 	"lastseen": lastseen,
			// 	"lux":      s[1],
			// 	"temp":     s[2],
			// 	"lstate":   s[0],
			//})
		} else {
			c.JSON(200, gin.H{
				"message": "No data retreived. Check Network Connection",
			})
		}
	})

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static")
	})

	r.StaticFS("/static", http.Dir("./web"))

	//MQTT

	// Create an MQTT Client.
	cli := client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})

	// Terminate the Client.
	defer cli.Terminate()

	defer cli.Disconnect()

	// Connect to the MQTT Server.
	err := cli.Connect(&client.ConnectOptions{
		Network:  "tcp",
		Address:  "192.168.2.37:1883",
		ClientID: []byte("example-client"),
	})
	if err != nil {
		fmt.Println("Error on mqtt connect:", err)
		os.Exit(1)
	}

	go mqttWorker(cli)

	r.Run() // listen and server on 0.0.0.0:8080
}
