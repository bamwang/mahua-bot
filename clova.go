package main

import (
	"net/url"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
)

func forwardToClova(event *linebot.Event, messages []linebot.Message) []linebot.Message {
	if message, ok := event.Message.(*linebot.TextMessage); ok {
		args := strings.Split(message.Text, " ")
		if len(args) < 3 {
			messages = append(messages, linebot.NewTextMessage("请发送\nclova 日语内容"))
			return messages
		}

		values := url.Values{}
		values.Set("message", args[1])
		messages = append(messages, linebot.NewTextMessage(post("https://bam-hook.herokuapp.com/clova", values.Encode())))
	}
	return messages
}
