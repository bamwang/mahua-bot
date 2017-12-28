package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	apiURL = "https://eth.nanopool.org/api/v1/load_account"
)

type HashData struct {
	UserParams  UserParams  `json:userParams`
	AvgHashRate AvgHashRate `json:avgHashRate`
}

type UserParams struct {
	HashRate  float32 `json:hashrate`
	Reported  float32 `json:reported`
	Balance   float32 `json:balance`
	TotalPaid float32 `json:e_sum`
}

type AvgHashRate struct {
	H24 string `json:h24`
	H12 string `json:h12`
	H6  string `json:h6`
	H3  string `json:h3`
	H1  string `json:h1`
}

type HashRes struct {
	Status string   `json:"status"`
	Data   HashData `json:"data"`
}

func getMiningStatus(event *linebot.Event, messages []linebot.Message, address, dig string) []linebot.Message {
	println("hash")
	if _, ok := event.Message.(*linebot.TextMessage); ok {
		messages = append(messages, linebot.NewTextMessage(do(address, dig)))
	}
	return messages
}

func do(address, dig string) (rep string) {
	url := fmt.Sprintf("%s/%s/%s", apiURL, address, dig)
	resp, err := http.Get(url)
	if err != nil {
		log.Print(err)
	}
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
	}
	var respStruct HashRes
	log.Println(url)
	log.Println(string(resBody))
	if err := json.Unmarshal(resBody, &respStruct); err != nil {
		log.Print(err)
	}
	data := respStruct.Data
	rep += fmt.Sprintf("address  : %s\n", address)
	rep += fmt.Sprintf("hashrate : %f\n", data.UserParams.HashRate)
	rep += fmt.Sprintf("reported : %f\n", data.UserParams.Reported)
	rep += fmt.Sprintf("balance  : %f\n", data.UserParams.Balance)
	rep += fmt.Sprintf("total    : %f\n", data.UserParams.TotalPaid)
	rep += fmt.Sprintln("=================")
	rep += fmt.Sprintf("h1  : %s\n", data.AvgHashRate.H1)
	rep += fmt.Sprintf("h3  : %s\n", data.AvgHashRate.H3)
	rep += fmt.Sprintf("h6  : %s\n", data.AvgHashRate.H6)
	rep += fmt.Sprintf("h12 : %s\n", data.AvgHashRate.H12)
	rep += fmt.Sprintf("h24 : %s\n", data.AvgHashRate.H24)
	log.Println(rep)
	return rep
}
