package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	tulingURL = "http://www.tuling123.com/openapi/api"
	tulingKey = os.Getenv("TULING_KEY")
)

type TulingResp struct {
	Code int64
	Text string
	Url  string
}

type TulingReq struct {
	Key    string `json:"key"`
	Info   string `json:"info"`
	UserID string `json:"userid"`
}

func forwardToTuling(event linebot.Event, messages []linebot.Message) []linebot.Message {
	if message, ok := event.Message.(*linebot.TextMessage); ok {
		hasher := sha1.New()
		hasher.Write([]byte(event.Source.UserID + event.Source.GroupID + event.Source.RoomID))
		id := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
		messages = append(messages, linebot.NewTextMessage(tulingDo(id, message.Text)))
	}
	return messages
}

func tulingDo(id, info string) (rep string) {
	b, _ := json.Marshal(TulingReq{tulingKey, info, id})
	resp, err := http.Post(tulingURL, "application/json", strings.NewReader(string(b)))
	if err != nil {
		log.Print(err)
	}
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
	}
	var respStruct TulingResp
	if err := json.Unmarshal(resBody, &respStruct); err != nil {
		log.Print(err)
	}
	rep = respStruct.Text
	if respStruct.Url != "" {
		rep += respStruct.Url
	}
	log.Println(rep)
	return rep
}
