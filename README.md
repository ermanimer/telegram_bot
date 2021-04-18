# telegram_bot
Go Telegram Bot

![Go](https://github.com/ermanimer/telegram_bot/workflows/Go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/ermanimer/telegram_bot)](https://goreportcard.com/report/github.com/ermanimer/telegram_bot)

## Features
telegram_bot gets updates of telegram bot, sends message to telegram bot's chats.

## Telegram Bots
Telegram bots can be created with [botfather](https://t.me/botfather).

## Sample Application
Sample application creates and starts a bot. Listens bot's updates. Sends "hello" message for every "/start" message.
```go
package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/ermanimer/telegram_bot/v2"
)

//provide your own token
const (
	token    = "..."
	interval = 2 * time.Second
	timeout  = 10 * time.Second
)

func main() {
	//create new bot
	b := telegram_bot.New(token, interval, timeout)
	//create channel for interupt signal
	is := make(chan os.Signal, 1)
	signal.Notify(is, os.Interrupt)
	//create done channel
	d := make(chan struct{})
	//start bot and listen bot's updates
	go func() {
		b.Start()
		for {
			select {
			case u := <-b.Updates:
				//check if updates are ok
				if !u.Ok {
					log.Println("updates are not ok")
					continue
				}
				//check if there is new updates
				if len(u.Result) == 0 {
					continue
				}
				//say hello when "/start" message is received
				for _, r := range u.Result {
					if r.Message.Text == "/start" {
						_, err := b.SendMessage(r.Message.Chat.ID, "hello")
						if err != nil {
							log.Printf("sending message failed: %s", err.Error())
							continue
						}
					}
				}
			case err := <-b.Error:
				//log error
				log.Printf("getting updates failed: %s", err.Error())
			case <-d:
				//end goroutine
				return
			}
		}
	}()
	//wait for interupt signal
	<-is
	//stop bot
	b.Stop()
	//close channels
	close(is)
	close(d)
}
```

## UpdatesResponse
```go
type UpdatesResponse struct {
	Ok     bool `json:"ok"`
	Result []struct {
		UpdateID int `json:"update_id"`
		Message  struct {
			MessageID int `json:"message_id"`
			From      struct {
				ID           int    `json:"id"`
				IsBot        bool   `json:"is_bot"`
				FirstName    string `json:"first_name"`
				LastName     string `json:"last_name"`
				LanguageCode string `json:"language_code"`
			} `json:"from"`
			Chat struct {
				ID        int    `json:"id"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
				Type      string `json:"type"`
			} `json:"chat"`
			Date     int    `json:"date"`
			Text     string `json:"text"`
			Entities []struct {
				Offset int    `json:"offset"`
				Length int    `json:"length"`
				Type   string `json:"type"`
			} `json:"entities"`
		} `json:"message"`
	} `json:"result"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}
```

## SendMessageResponse
```go
type MessageResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID        int    `json:"id"`
			IsBot     bool   `json:"is_bot"`
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Type      string `json:"type"`
		} `json:"chat"`
		Date int    `json:"date"`
		Text string `json:"text"`
	} `json:"result"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}
```
