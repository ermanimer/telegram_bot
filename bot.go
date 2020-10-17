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

type Bot struct {
	input          chan *input
	Output         chan *output
	token          string
	chats          map[int]bool
	isStarted      bool
	getUpdatesUrl  string
	sendMessageUrl string
	interval       time.Duration
	offset         int
	mutex          *sync.Mutex
}

//chats file
const (
	chatsFile         = "./chats.json"
	chatsFileFileMode = 0644
)

//telegram api url templates
const (
	getUpdatesUrlTemplate  = "https://api.telegram.org/bot<token>/getUpdates"
	sendMessageUrlTemplate = "https://api.telegram.org/bot<token>/sendMessage"
)

//update commands
const (
	startCommand = "/start"
	stopCommand  = "/stop"
)

func NewBot(token string, interval int) *Bot {
	//create a new instance
	b := &Bot{
		input:          make(chan *input),
		Output:         make(chan *output),
		token:          token,
		chats:          make(map[int]bool),
		getUpdatesUrl:  strings.ReplaceAll(getUpdatesUrlTemplate, "<token>", token),
		sendMessageUrl: strings.ReplaceAll(sendMessageUrlTemplate, "<token>", token),
		interval:       time.Duration(interval) * time.Millisecond,
		mutex:          &sync.Mutex{},
	}
	//listen input channel
	go func() {
		for {
			i := <-b.input
			//receive start signal to start getting chat updates
			if i.Start {
				if b.isStarted {
					b.sendErrorMessage("bot is already started")
					continue
				}
				go b.startGettingUpdates()
				continue
			}
			//receive stop signal to stop getting chat updates
			if i.Stop {
				if !b.isStarted {
					b.sendErrorMessage("bot is already stopped or not initialized")
					continue
				}
				b.stopGettingUpdates()
				continue
			}
			//receive a message to send to all active chats
			if i.Message != "" {
				if !b.isStarted {
					b.sendErrorMessage("bot is already stopped or not initialized")
				}
				if len(b.chats) == 0 {
					b.sendErrorMessage("bot doesn't have any chats")
				}
				b.sendMessageToAllActiveChats(i.Message)
			}
		}
	}()
	//return created instance
	return b
}

//starts getting updates
func (b *Bot) Start() {
	b.input <- &input{
		Start: true,
	}
}

//stops getting updates
func (b *Bot) Stop() {
	b.input <- &input{
		Stop: true,
	}
}

//sends message to all active chats
func (b *Bot) SendMessage(messages ...interface{}) {
	messageFormat := createMessageFormat(len(messages))
	b.input <- &input{
		Message: fmt.Sprintf(messageFormat, messages...),
	}
}

//sends formatted message to all active chats
func (b *Bot) SendMessagef(messageFormat string, messages ...interface{}) {
	b.input <- &input{
		Message: fmt.Sprintf(messageFormat, messages...),
	}
}

//gets chat updates regularly with using an sleep interval
func (b *Bot) startGettingUpdates() {
	b.isStarted = true
	b.sendInfoMessage("bot is started")
	err := b.loadChats()
	if err != nil {
		b.sendErrorMessage(err.Error())
	}
	for {
		time.Sleep(b.interval)
		if !b.isStarted {
			break
		}
		r, err := b.getUpdates()
		if err != nil {
			b.sendErrorMessage(err.Error())
			continue
		}
		if !r.Ok {
			b.sendErrorMessagef("getting updates failed error code: %v description: %v", r.ErrorCode, r.Description)
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
			b.sendInfoMessagef("%v command received from chat id: %v first name: %v last name: %v", command, chatId, firstName, lastName)
			b.offset = updateId + 1
		}
		err = b.saveChats()
		if err != nil {
			b.sendErrorMessage(err.Error())
		}
	}
	b.sendInfoMessage("bot is stopped")
}

//stops getting chat updates
func (b *Bot) stopGettingUpdates() {
	b.isStarted = false
}

//sends message to all active chats synchronously
func (b *Bot) sendMessageToAllActiveChats(message string) {
	for chatId, isStarted := range b.chats {
		if isStarted {
			r, err := b.sendMessage(chatId, message)
			if err != nil {
				b.sendErrorMessage(err.Error())
				continue
			}
			if !r.Ok {
				b.sendErrorMessagef("sending message failed to chat id: %v error code: %v description: %v", chatId, r.ErrorCode, r.Description)
			}
		}
	}
}

//gets chat updates
func (b *Bot) getUpdates() (*getUpdatesResponse, error) {
	gureq := getUpdatesRequest{
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
	var gures getUpdatesResponse
	err = json.NewDecoder(res.Body).Decode(&gures)
	if err != nil {
		return nil, errors.New("unmarshalling get updates response failed")
	}
	return &gures, nil
}

//sends message to an active chat
func (b *Bot) sendMessage(chatId int, message string) (*sendMessageResponse, error) {
	smreq := sendMessageRequest{
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
	var smres sendMessageResponse
	err = json.NewDecoder(res.Body).Decode(&smres)
	if err != nil {
		return nil, errors.New("unmarshalling send message response failed")
	}
	return &smres, nil
}

//loads chats records from chats file
func (b *Bot) loadChats() error {
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

//saves chat records to chats file
func (b *Bot) saveChats() error {
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

//sends info message through output channel
func (b *Bot) sendInfoMessage(messages ...interface{}) {
	messageFormat := createMessageFormat(len(messages))
	b.Output <- &output{
		InfoMessage: fmt.Sprintf(messageFormat, messages...),
	}
}

//sends formatted info message output input channel
func (b *Bot) sendInfoMessagef(messageFormat string, messages ...interface{}) {
	b.Output <- &output{
		InfoMessage: fmt.Sprintf(messageFormat, messages...),
	}
}

//sends error message through output channel
func (b *Bot) sendErrorMessage(messages ...interface{}) {
	messageFormat := createMessageFormat(len(messages))
	b.Output <- &output{
		ErrorMessage: fmt.Sprintf(messageFormat, messages...),
	}
}

//sends formatted error message through output channel
func (b *Bot) sendErrorMessagef(messageFormat string, messages ...interface{}) {
	b.Output <- &output{
		ErrorMessage: fmt.Sprintf(messageFormat, messages...),
	}
}

//creates message format for message functions
func createMessageFormat(messageCount int) string {
	messageFormat := strings.Repeat("%v, ", messageCount)
	messageFormat = strings.Trim(messageFormat, ", ")
	return messageFormat
}
