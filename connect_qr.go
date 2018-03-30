package main

import (
	"net/url"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
)

func forwardToQR(event *linebot.Event, messages []linebot.Message) []linebot.Message {
	if message, ok := event.Message.(*linebot.TextMessage); ok {
		args := strings.Split(message.Text, " ")
		if len(args) < 3 {
			messages = append(messages, linebot.NewTextMessage("请发送\nqr 社员号 密码 自己的邮箱地址\n邮箱地址不填写则只发送到公司邮箱"))
			return messages
		}

		values := url.Values{}
		values.Set("username", args[1])
		values.Set("password", args[2])
		if len(args) == 4 {
			values.Set("privateMail", args[3])
		}
		messages = append(messages, linebot.NewTextMessage(post("https://bam-hook.herokuapp.com/connect_qr", values.Encode())))
	}
	return messages
}
