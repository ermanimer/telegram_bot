package telegram_bot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	chatsFile         = "./chats.json"
	chatsFileFileMode = 0644
)

const (
	getUpdatesUrlTemplate  = "https://api.telegram.org/bot<token>/getUpdates"
	sendMessageUrlTemplate = "https://api.telegram.org/bot<token>/sendMessage"
)

const (
	startCommand = "/start"
	stopCommand  = "/stop"
)

var b *Bot

func Initialize(token string, interval int) *Bot {
	if b == nil {
		b = &Bot{
			Output:         make(chan *Output),
			token:          token,
			chats:          make(map[int]bool),
			getUpdatesUrl:  strings.ReplaceAll(getUpdatesUrlTemplate, "<token>", token),
			sendMessageUrl: strings.ReplaceAll(sendMessageUrlTemplate, "<token>", token),
			interval:       time.Duration(interval) * time.Millisecond,
			mutex:          &sync.Mutex{},
		}
	}
	return b
}

func (b *Bot) Start() error {
	if b == nil {
		return errors.New("bot is not initialized")
	}
	if b.isStarted {
		return errors.New("bot is already started")
	}
	go func() {
		b.isStarted = true
		publishInfoMessage("bot is started")
		err := loadChats()
		if err != nil {
			publishErrorMessage(err.Error())
		}
		for {
			time.Sleep(b.interval)
			if !b.isStarted {
				break
			}
			r, err := getUpdates()
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			if !r.Ok {
				publishErrorMessage(fmt.Sprintf("getting updates failed error code: %v description: %v", r.ErrorCode, r.Description))
				continue
			}
			us := r.Result
			if len(us) == 0 {
				continue
			}
			for _, u := range us {
				updateId := u.UpdateId
				command := u.Message.Text
				chatId := u.Message.Chat.Id
				firstName := u.Message.From.FirstName
				lastName := u.Message.From.LastName
				switch command {
				case startCommand:
					b.chats[chatId] = true
				case stopCommand:
					b.chats[chatId] = false
				}
				publishInfoMessage(fmt.Sprintf("%v command received from chat id: %v first name: %v last name: %v", command, chatId, firstName, lastName))
				b.offset = updateId + 1
			}
			err = updateChats()
			if err != nil {
				publishErrorMessage(err.Error())
			}
		}
		publishInfoMessage("bot is stopped")
	}()
	return nil
}

func (b *Bot) Stop() error {
	if !b.isStarted {
		return errors.New("bot is already stopped or not initialized")
	}
	b.isStarted = false
	return nil
}

func (b *Bot) SendMessage(message string) error {
	if !b.isStarted {
		return errors.New("bot is already stopped or not initialized")
	}
	if len(b.chats) == 0 {
		return errors.New("bot doesn't have any chats")
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for chatId, isStarted := range b.chats {
		if isStarted {
			r, err := sendMessage(chatId, message)
			if err != nil {
				publishErrorMessage(err.Error())
				continue
			}
			if !r.Ok {
				publishErrorMessage(fmt.Sprintf("sending message failed to chat id: %v error code: %v description: %v", chatId, r.ErrorCode, r.Description))
			}
		}
	}
	return nil
}

func getUpdates() (*GetUpdatesResponse, error) {
	gureq := GetUpdatesRequest{
		Offset: b.offset,
	}
	reqb, err := json.Marshal(&gureq)
	if err != nil {
		return nil, errors.New("marshalling get updates request body failed")
	}
	req, err := http.NewRequest("POST", b.getUpdatesUrl, bytes.NewBuffer(reqb))
	if err != nil {
		return nil, errors.New("creating get updates request body failed")
	}
	req.Header.Set("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.New("doing get updates http request failed")
	}
	defer res.Body.Close()
	var gures GetUpdatesResponse
	err = json.NewDecoder(res.Body).Decode(&gures)
	if err != nil {
		return nil, errors.New("unmarshalling get updates response failed")
	}
	return &gures, nil
}

func sendMessage(chatId int, message string) (*SendMessageResponse, error) {
	smreq := SendMessageRequest{
		ChatId: chatId,
		Text:   message,
	}
	reqb, err := json.Marshal(&smreq)
	if err != nil {
		return nil, errors.New("marshalling send message request body failed")
	}
	req, err := http.NewRequest("POST", b.sendMessageUrl, bytes.NewBuffer(reqb))
	if err != nil {
		return nil, errors.New("creating send message http request failed")
	}
	req.Header.Set("Content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.New("doing send message http request failed")
	}
	defer res.Body.Close()
	var smres SendMessageResponse
	err = json.NewDecoder(res.Body).Decode(&smres)
	if err != nil {
		return nil, errors.New("unmarshalling send message response failed")
	}
	return &smres, nil
}

func loadChats() error {
	f, err := os.Open(chatsFile)
	if err != nil {
		return errors.New("opening chats file failed")
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&b.chats)
	if err != nil {
		return errors.New("decoding chats file failed")
	}
	return nil
}

func updateChats() error {
	f, err := os.OpenFile(chatsFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, chatsFileFileMode)
	if err != nil {
		return errors.New("opening chats file failed")
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(b.chats)
	if err != nil {
		return errors.New("encoding chats failed")
	}
	return nil
}

func publishInfoMessage(message string) {
	b.Output <- &Output{
		InfoMessage: message,
	}
}

func publishErrorMessage(message string) {
	b.Output <- &Output{
		ErrorMessage: message,
	}
}
