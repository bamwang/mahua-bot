package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
)

func forwardToMsgc(event *linebot.Event, messages []linebot.Message) []linebot.Message {
	values := url.Values{}
	if event != nil {
		if message, ok := event.Message.(*linebot.TextMessage); ok {
			args := strings.Split(message.Text, " ")
			if len(args) == 3 {
				values.Set("username", args[1])
				values.Set("password", args[2])
			}
		}
	}
	msgs := post("https://bam-hook.herokuapp.com/connect_msg_check", values.Encode())
	var m []map[string]interface{}
	err := json.NewDecoder(strings.NewReader(msgs)).Decode(&m)
	if err != nil {
		log.Println(err)
	}
	var rep string
	if len(m) == 0 {
		rep = "残念！"
		return append(messages, linebot.NewTextMessage(rep))
	}
	rep = "有空哦！\n"
	var firstDate string
	for _, msg := range m {
		var massagerNames []string
		for _, massager := range msg["massagers"].([]interface{}) {
			massagerNames = append(massagerNames, massager.(map[string]interface{})["name"].(string))
		}
		rep += fmt.Sprintf("%s@%s by %s\n", msg["time"].(string)[0:2], strings.Join(strings.Split(msg["date"].(string), ".")[1:], "."), strings.Join(massagerNames, ","))
		if firstDate == "" {
			firstDate = msg["date"].(string)
		}
	}
	if len(firstDate) > 10 {
		rep += "http://connect.navercorp.com/reserve/healthServiceList.nhn?reserveCurrentDate=" + firstDate[:10]
	}
	return append(messages, linebot.NewTextMessage(rep))
}
