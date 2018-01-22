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
	generalInfoURL = "https://api.nanopool.org/v1/eth/user"
	reportedURL    = "https://api.nanopool.org/v1/eth/reportedhashrates"
	averageURL     = "https://api.nanopool.org/v1/eth/avghashrateworkers"
)

type Res interface {
	// GetData() interface{}
	GetStatus() bool
}
type ReportedHashrateRes struct {
	Status bool                   `json:status`
	Data   []ReportedHashrateData `json:data`
}

type ReportedHashrateData struct {
	Worker   string  `json:worker`
	Hashrate float32 `json:hashrate`
}

type GeneralInfoRes struct {
	Status bool            `json:status`
	Data   GeneralInfoData `json:data`
}

type GeneralInfoData struct {
	Account            string `json:account`
	Balance            string `json:balance`
	UnconfirmedBalance string `json:unconfirmed_balance`
	Hashrate           string `json:hashrate`
	AvgHashRate        struct {
		H24 string `json:h24`
		H12 string `json:h12`
		H6  string `json:h6`
		H3  string `json:h3`
		H1  string `json:h1`
	} `json:avgHashRate`
}

type AvgHashrateWorkersRes struct {
	Status bool                   `json:status`
	Data   AvgHashrateWorkersData `json:data`
}
type AvgHashrateWorkersData struct {
	H24 []AvgHashrateWorkersDataDetail `json:h24`
	H12 []AvgHashrateWorkersDataDetail `json:h12`
	H6  []AvgHashrateWorkersDataDetail `json:h6`
	H3  []AvgHashrateWorkersDataDetail `json:h3`
	H1  []AvgHashrateWorkersDataDetail `json:h1`
}

type AvgHashrateWorkersDataDetail struct {
	Worker   string  `json:worker`
	Hashrate float32 `json:hashrate`
}

func (res *GeneralInfoRes) GetStatus() bool {
	return res.Status
}

func (res *ReportedHashrateRes) GetStatus() bool {
	return res.Status
}

func (res *AvgHashrateWorkersRes) GetStatus() bool {
	return res.Status
}

func getMiningStatus(event *linebot.Event, messages []linebot.Message, address, dig string) []linebot.Message {
	println("hash")
	if _, ok := event.Message.(*linebot.TextMessage); ok {
		messages = append(messages, linebot.NewTextMessage(do(address)))
	}
	return messages
}

func call(baseURL, address string, res Res) (Res, bool) {
	url := fmt.Sprintf("%s/%s", baseURL, address)
	resp, err := http.Get(url)
	if err != nil {
		log.Print(err)
	}
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
	}
	log.Println(url)
	log.Println(string(resBody))
	if err := json.Unmarshal(resBody, &res); err != nil {
		log.Print(err)
	}
	return res, res.GetStatus()
}

func getGeneralInfo(address string) (GeneralInfoData, bool) {
	generalInfoURL := fmt.Sprintf("%s/%s", generalInfoURL, address)
	resp, err := http.Get(generalInfoURL)
	if err != nil {
		log.Print(err)
	}
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
	}
	var respStruct GeneralInfoRes
	log.Println(generalInfoURL)
	log.Println(string(resBody))
	if err := json.Unmarshal(resBody, &respStruct); err != nil {
		log.Print(err)
	}
	return respStruct.Data, respStruct.Status
}

func getReportedHashrates(address string) ([]ReportedHashrateData, bool) {
	reportedURL := fmt.Sprintf("%s/%s", reportedURL, address)
	resp, err := http.Get(reportedURL)
	if err != nil {
		log.Print(err)
	}
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
	}
	var respStruct ReportedHashrateRes
	log.Println(reportedURL)
	log.Println(string(resBody))
	if err := json.Unmarshal(resBody, &respStruct); err != nil {
		log.Print(err)
	}
	return respStruct.Data, respStruct.Status
}

func do(address string) (rep string) {
	rep += fmt.Sprintf("address  : %s\n", address)
	{
		var res GeneralInfoRes
		call(generalInfoURL, address, &res)
		if res.Status == true {
			rep += fmt.Sprintf("hashrate : %s\n", res.Data.Hashrate)
			rep += fmt.Sprintf("balance  : %s\n", res.Data.Balance)
			rep += fmt.Sprintln("======== Total =========")
			rep += fmt.Sprintf("h1       : %s\n", res.Data.AvgHashRate.H1)
			rep += fmt.Sprintf("h3       : %s\n", res.Data.AvgHashRate.H3)
			rep += fmt.Sprintf("h6       : %s\n", res.Data.AvgHashRate.H6)
			rep += fmt.Sprintf("h12      : %s\n", res.Data.AvgHashRate.H12)
			rep += fmt.Sprintf("h24      : %s\n", res.Data.AvgHashRate.H24)
		}
	}
	{
		var reportRes ReportedHashrateRes
		var avgRes AvgHashrateWorkersRes
		// var res
		call(reportedURL, address, &reportRes)
		call(averageURL, address, &avgRes)
		type MappedHashrate struct {
			AvgH1    float32
			AvgH6    float32
			Reported float32
		}
		m := map[string]MappedHashrate{}
		if reportRes.Status == true {
			for _, worker := range reportRes.Data {
				m[worker.Worker] = MappedHashrate{
					Reported: worker.Hashrate,
				}
			}
			for _, worker := range avgRes.Data.H1 {
				if w, has := m[worker.Worker]; has {
					w.AvgH1 = worker.Hashrate
					m[worker.Worker] = w
				}
			}
			for _, worker := range avgRes.Data.H6 {
				if w, has := m[worker.Worker]; has {
					w.AvgH6 = worker.Hashrate
					m[worker.Worker] = w
				}
			}
		}
		for name, hashrate := range m {
			rep += fmt.Sprintln("======== " + name + " =========")
			rep += fmt.Sprintf("reported : %6.2f\n", hashrate.Reported)
			rep += fmt.Sprintf("h1       : %6.2f\n", hashrate.AvgH1)
			rep += fmt.Sprintf("h6       : %6.2f\n", hashrate.AvgH6)
		}
	}
	log.Println(rep)
	return rep
}
