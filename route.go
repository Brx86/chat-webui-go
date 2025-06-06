package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

type C = *gin.Context

func setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Static("/static", "static")
	r.StaticFile("/", "static/app.html")
	r.GET("/fetch-models", func(c C) {
		c.JSON(200, preloadModels)
	})
	r.POST("/save-settings", func(c C) {
		if err := c.ShouldBindJSON(&cfg); err != nil {
			c.String(400, err.Error())
			return
		}
		// saveConfig()
		initModels()
		c.JSON(200, gin.H{"status": "success"})
	})
	r.POST("/chat", handleStream)
	r.POST("/continue_generation", handleStream)
	r.POST("/generate-title", func(c C) {
		data := new(ChatData)
		if err := c.ShouldBindJSON(&data); err != nil {
			c.String(400, err.Error())
			return
		}
		title := genTitle(data)
		c.JSON(200, gin.H{"title": title})
	})
	return r
}

func handleStream(c C) {
	data := new(ChatData)
	if err := c.ShouldBindJSON(data); err != nil {
		c.String(400, err.Error())
		return
	}
	if len(data.Message) > 0 {
		err := json.Unmarshal(data.Message, &data.MessageText)
		if err == nil {
			log.Println("Message is a string")
		} else {
			err = json.Unmarshal(data.Message, &data.MessagePart)
			if err == nil && len(data.MessagePart) > 0 {
				log.Println("Message with image", data.MessagePart[0].Text)
			}
		}
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	streamCh := make(chan string)
	go generate(streamCh, data)
	for s := range streamCh {
		fmt.Fprint(c.Writer, s)
		c.Writer.Flush()
	}
}
