package telegram_bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

//telegram api url templates
const (
	getUpdatesURLTemplate  = "https://api.telegram.org/bot<token>/getUpdates"
	sendMessageURLTemplate = "https://api.telegram.org/bot<token>/sendMessage"
)

//Bot represent telegram bot
type Bot struct {
	Token          string
	Interval       time.Duration         //interval between continous get updates requests
	Timeout        time.Duration         //http client timeout for get updates and send message requests
	Updates        chan *UpdatesResponse //channel for publishing get updates responses
	Error          chan error            //channel for publishing get updates reqeust errors
	t              *time.Ticker          //ticker for get updates requests
	d              chan struct{}         //done channel to stop get updates loop
	offset         int                   //offset for get updates request, to filter out previously received updates
	getUpdatesURL  string                //url for get updates request
	sendMessageURL string                //url for send message request
}

//New, creates new bot
func New(token string, interval, timeout int) *Bot {
	return &Bot{
		Token:          token,
		Interval:       time.Duration(interval) * time.Millisecond,
		Updates:        make(chan *UpdatesResponse),
		Error:          make(chan error),
		getUpdatesURL:  strings.Replace(getUpdatesURLTemplate, "<token>", token, 1),
		sendMessageURL: strings.Replace(sendMessageURLTemplate, "<token>", token, 1),
	}
}

//Start starts bot.
func (tb *Bot) Start() {
	//initialize ticker and done channel
	tb.t = time.NewTicker(tb.Interval)
	tb.d = make(chan struct{})
	//start
	go func() {
		defer func() {
			close(tb.Updates)
			close(tb.Error)
		}()
		for {
			select {
			case <-tb.t.C:
				u, err := tb.getUpdates()
				if err != nil {
					tb.Error <- fmt.Errorf("getting updates failed: %s", err.Error())
					continue
				}
				tb.Updates <- u
			case <-tb.d:
				return
			}
		}
	}()
}

//Stop, stops bot
func (tb *Bot) Stop() {
	//stop ticker
	tb.t.Stop()
	//stop
	close(tb.d)
}

//SendMessage sends a message to chat which is defined by chatId.
func (tb *Bot) SendMessage(chatId int, message string) (*MessageResponse, error) {
	//create message request
	mreq := messageRequest{
		ChatId: chatId,
		Text:   message,
	}
	//create http request body
	reqb, err := json.Marshal(&mreq)
	if err != nil {
		return nil, fmt.Errorf("encoding http request body failed: %s", err.Error())
	}
	//create http request
	req, err := http.NewRequest("POST", tb.sendMessageURL, bytes.NewBuffer(reqb))
	if err != nil {
		return nil, fmt.Errorf("creating http request failed: %s", err.Error())
	}
	//set request headers
	req.Header.Set("Content-type", "application/json")
	//create http client
	c := http.Client{
		Timeout: tb.Timeout,
	}
	//do http request
	res, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %s", err.Error())
	}
	defer res.Body.Close()
	//decode message response
	var mres MessageResponse
	err = json.NewDecoder(res.Body).Decode(&mres)
	if err != nil {
		return nil, fmt.Errorf("decoding message response failed: %s", err.Error())
	}
	//return updates response
	return &mres, nil
}

//getUpdates posts a http request to get updates from telegram api.
func (tb *Bot) getUpdates() (*UpdatesResponse, error) {
	//create updates request
	ureq := updatesRequest{
		Offset: tb.offset + 1,
	}
	//create http request body
	reqb, err := json.Marshal(&ureq)
	if err != nil {
		return nil, fmt.Errorf("encoding http request body failed: %s", err.Error())
	}
	//create http request
	req, err := http.NewRequest("POST", tb.getUpdatesURL, bytes.NewBuffer(reqb))
	if err != nil {
		return nil, fmt.Errorf("creating http request failed: %s", err.Error())
	}
	//set request headers
	req.Header.Set("Content-type", "application/json")
	//create http client
	c := http.Client{
		Timeout: tb.Timeout,
	}
	//do http request
	res, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %s", err.Error())
	}
	defer res.Body.Close()
	//decode updates response
	var ures UpdatesResponse
	err = json.NewDecoder(res.Body).Decode(&ures)
	if err != nil {
		return nil, fmt.Errorf("decoding updates response failed: %s", err.Error())
	}
	//update offset
	tb.updateOffset(&ures)
	//return updates response
	return &ures, nil
}

//updateOffset updates offset using latest update id of updates response.
func (tb *Bot) updateOffset(ur *UpdatesResponse) {
	//return if updates response is not ok or there is no new result
	if !ur.Ok || len(ur.Result) == 0 {
		return
	}
	//set offset to the latest update id
	for _, r := range ur.Result {
		if r.UpdateId > tb.offset {
			tb.offset = r.UpdateId
		}
	}
}
