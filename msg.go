package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
	mgo "gopkg.in/mgo.v2"
)

var massageRemaineds = make(map[string][]*linebot.CarouselColumn)

func createMassageInfo(massages *mgo.Collection) linebot.Message {
	blocks := Blocks{}
	err := massages.Find(nil).Sort("-_id").One(&blocks)
	if err != nil && err != mgo.ErrNotFound {
		log.Fatalln(err)
	}
	var rep string
	if len(blocks.Blocks) == 0 {
		rep = "残念！"
		return linebot.NewTextMessage(rep)
	}
	rep = "有空哦！\n"
	var firstDate string
	for _, block := range blocks.Blocks {
		rep += fmt.Sprintf("%s@%s by %s\n", block.Time[0:2], strings.Join(strings.Split(block.Date, ".")[1:], "."), strings.Join(block.Massagers, ","))
		if firstDate == "" {
			firstDate = block.Date
		}
	}
	if len(firstDate) > 10 {
		rep += "http://connect.navercorp.com/reserve/healthServiceList.nhn?reserveCurrentDate=" + firstDate[:10]
	}
	return linebot.NewTextMessage(rep)
}

func createMassageMessage(id string, massages *mgo.Collection) linebot.Message {
	cols := make([]*linebot.CarouselColumn, 0)
	if remainedCols, has := massageRemaineds[id]; has && len(remainedCols) > 0 {
		cols = remainedCols
	} else {
		delete(massageRemaineds, id)
		blocks := Blocks{}
		err := massages.Find(nil).Sort("-_id").One(&blocks)
		if err != nil && err != mgo.ErrNotFound {
			log.Fatalln(err)
		}
		dateMap := make(map[string][]Block)
		if len(blocks.Blocks) == 0 {
			return linebot.NewTextMessage("残念")
		}
		for _, block := range blocks.Blocks {
			if _, has := dateMap[block.Date]; !has {
				dateMap[block.Date] = make([]Block, 0)
			}
			dateMap[block.Date] = append(dateMap[block.Date], block)
		}

		dateArr := make([]string, 0, len(dateMap))
		for date := range dateMap {
			dateArr = append(dateArr, date)
		}

		sort.Strings(dateArr)

		for _, date := range dateArr {
			blocks := dateMap[date]
			text := ""
			for _, block := range blocks {
				if len(text) > 50 {
					text += "..."
					break
				}
				text += fmt.Sprintf("%s:%s\n", block.Time[0:2], strings.Join(block.Massagers, ","))
			}
			url := "http://connect.navercorp.com/reserve/healthServiceList.nhn?reserveCurrentDate=" + date[:10]
			cols = append(cols, linebot.NewCarouselColumn("", "", date+"\n"+text, linebot.NewURITemplateAction("BOOK NOW!", url)))
		}
	}
	max := 5
	if len(cols) > 5 {
		massageRemaineds[id] = cols[5:]
		// cols[4].Actions = append(cols[len(cols)-1].Actions, linebot.NewMessageTemplateAction("SEE MORE", "msg"))
		cols[4].Text += "\nand more dates"
	} else {
		max = len(cols)
		delete(massageRemaineds, id)
	}

	return linebot.NewTemplateMessage("Massage", linebot.NewCarouselTemplate(cols[:max]...))
}
