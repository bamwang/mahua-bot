package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	endpoint = "https://bam-hook.herokuapp.com/e4628"
)

func forwardToBamhook(event *linebot.Event, messages []linebot.Message) []linebot.Message {
	println("tuling")
	if message, ok := event.Message.(*linebot.TextMessage); ok {
		args := strings.Split(message.Text, " ")
		if len(args) < 4 {
			messages = append(messages, linebot.NewTextMessage("请发送\n上班:\ndk 社员号 密码 1\n下班:\ndk 社员号 密码 2"))
			return messages
		}
		values := url.Values{}
		values.Set("username", args[1])
		values.Set("password", args[2])
		values.Set("type", args[3])
		messages = append(messages, linebot.NewTextMessage(post(values.Encode())))
	}
	return messages
}

func post(body string) string {
	resp, err := http.Post(endpoint, "application/x-www-form-urlencoded", strings.NewReader(body))
	if err != nil {
		log.Print(err)
	}
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
	}
	return string(resBody)
}
