package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	mgo "gopkg.in/mgo.v2"

	"bitbucket.com/wangzhucn/mahua-bot/action_dispatcher"
	"github.com/line/line-bot-sdk-go/linebot"
)

var moyu = os.Getenv("MOYU_ID")
var laosiji = os.Getenv("LOAIJI_ID")

func register(dispatcher *actionDispatcher.ActionDispatcher, massages *mgo.Collection) {
	// f23
	f23MenuHandler := func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		donburiCol := linebot.NewCarouselColumn(fmt.Sprintf("%s/don.jpg", staticsPrefix), "Donburi / Curry", "JPY300", linebot.NewURITemplateAction("BUY", "https://webpos.line.me/cafe/payment/c81e728d9d4c2f636f067f89cc14862c/reserve"))
		saladCol := linebot.NewCarouselColumn(fmt.Sprintf("%s/salad.jpg", staticsPrefix), "Salad", "JPY100", linebot.NewURITemplateAction("BUY", "https://webpos.line.me/cafe/payment/a87ff679a2f3e71d9181a67b7542122c/reserve"))
		bentoCol := linebot.NewCarouselColumn(fmt.Sprintf("%s/bento.jpg", staticsPrefix), "Bento", "JPY500", linebot.NewURITemplateAction("BUY", "https://webpos.line.me/cafe/payment/c4ca4238a0b923820dcc509a6f75849b/reserve"))
		soupCol := linebot.NewCarouselColumn(fmt.Sprintf("%s/soup.jpg", staticsPrefix), "Soup", "JPY300", linebot.NewURITemplateAction("BUY", "https://webpos.line.me/cafe/payment/eccbc87e4b5ce2fe28308fd9f2a7baf3/reserve"))
		messages = append(messages, linebot.NewTemplateMessage("23F cafe menu", linebot.NewCarouselTemplate(donburiCol, saladCol, bentoCol, soupCol)))
		return
	}
	dispatcher.RegisterWithType([]string{"23b"}, []linebot.EventSourceType{}, actionDispatcher.NewReplayAction(f23MenuHandler))

	// msg
	MsgHandler := func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		message := createMassageMessage(event.Source.GroupID+event.Source.RoomID, massages)
		messages = append(messages, message)
		return
	}
	dispatcher.RegisterWithType([]string{"msg"}, []linebot.EventSourceType{}, actionDispatcher.NewReplayAction(MsgHandler))

	// msgs
	MsgsHandler := func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		message := createMassageInfo(massages)
		messages = append(messages, message)
		return
	}
	dispatcher.RegisterWithType([]string{"msgs"}, []linebot.EventSourceType{}, actionDispatcher.NewReplayAction(MsgsHandler))

	// mahua gallery
	MGHandler := func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		mahuas := loadMahua()
		if len(mahuas) == 0 {
			return
		}
		mahuaOrigin := mahuas[rand.Intn(len(mahuas))]
		mahuaBase := mahuaOrigin[:len(mahuaOrigin)-4]
		messages = append(messages, linebot.NewImageMessage(bucketURLPrefix+mahuaOrigin, bucketURLPrefix+mahuaBase+"_thumbnail.jpg"))
		return
	}
	dispatcher.RegisterWithType([]string{"看麻花"}, []linebot.EventSourceType{}, actionDispatcher.NewReplayAction(MGHandler))

	// mahua gallery
	MGNHandler := func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		mahuas := loadMahua()
		if len(mahuas) == 0 {
			return
		}
		mahuaOrigin := mahuas[len(mahuas)-1]
		mahuaBase := mahuaOrigin[:len(mahuaOrigin)-4]
		messages = append(messages, linebot.NewImageMessage(bucketURLPrefix+mahuaOrigin, bucketURLPrefix+mahuaBase+"_thumbnail.jpg"))
		return
	}
	dispatcher.RegisterWithType([]string{"最新麻花"}, []linebot.EventSourceType{}, actionDispatcher.NewReplayAction(MGNHandler))
	// fan
	fanActivateAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		name := getUserName(event.Source.UserID)

		userMap := map[string]string{
			event.Source.UserID: name,
		}
		messages = append(messages, linebot.NewTextMessage("好的, "+name))
		context.SetData(userMap)
		return
	})

	fanInactiveAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("麻花也去觅食啦！喵~~"))
		return
	})

	fanUsualAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		userMap := context.GetData().(map[string]string)
		context.SetData(userMap)
		names := []string{}
		userIDs := []string{}
		for id, name := range userMap {
			names = append(names, name)
			userIDs = append(userIDs, id)
		}
		if message, ok := event.Message.(*linebot.TextMessage); ok {
			name := getUserName(event.Source.UserID)
			switch message.Text {
			case "f":
				userMap[event.Source.UserID] = name
				context.SetData(userMap)
				messages = append(messages, linebot.NewTextMessage("好的, "+name))
			case "nf":
				delete(userMap, event.Source.UserID)
				context.SetData(userMap)
				messages = append(messages, linebot.NewTextMessage("拜拜, "+name))
			case "c":
				messages = append(messages, linebot.NewTextMessage(fmt.Sprintf("说好要去吃饭的有%d人：\n", len(userIDs))+strings.Join(names, "\n")))
			case "g":

				sentTo(userIDs, name+"喊道：走啦走啦！去吃饭啦！")
				messages = append(messages, linebot.NewTextMessage("走啦走啦：\n"+strings.Join(names, "\n")))
			}
		}
		return
	})

	dispatcher.RegisterWithType([]string{"f"}, []linebot.EventSourceType{linebot.EventSourceTypeGroup, linebot.EventSourceTypeRoom}, actionDispatcher.NewContextAction(
		[]string{"g!"},
		fanActivateAction,
		fanInactiveAction,
		fanUsualAction,
	))

	// mahua
	mahuaActivateAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("喵~~"))
		return
	})

	mahuaInactiveAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("汪~~"))
		return
	})

	mahuaUsualAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = forwardToTuling(event, messages)
		return
	})

	dispatcher.RegisterWithType([]string{"麻花来"}, []linebot.EventSourceType{linebot.EventSourceTypeGroup, linebot.EventSourceTypeRoom}, actionDispatcher.NewContextAction(
		[]string{"麻花拜拜"},
		mahuaActivateAction,
		mahuaInactiveAction,
		mahuaUsualAction,
	))

	// futi
	futiActivateAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("麻花为你代言"))
		return
	})

	futiInactiveAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("麻花还是麻花"))
		return
	})

	futiUsualAction := actionDispatcher.NewPushAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, id string, err error) {
		switch message := event.Message.(type) {
		case *linebot.StickerMessage:
			if i, err := strconv.Atoi(message.PackageID); err != nil || i > 4 {
				return messages, id, nil
			}
		case *linebot.ImageMessage:
			if originURL, thumbnailURL, err := fetchAndUploadContent(message.ID, "imgs"); err == nil {
				messages = append(messages, linebot.NewImageMessage(originURL, thumbnailURL))
			} else {
				log.Println(err)
			}
		case *linebot.TextMessage:
			messages = append(messages, event.Message)
		}
		id = moyu
		return
	})

	dispatcher.RegisterWithType([]string{"ft"}, []linebot.EventSourceType{linebot.EventSourceTypeUser}, actionDispatcher.NewContextAction(
		[]string{"lt"},
		futiActivateAction,
		futiInactiveAction,
		futiUsualAction,
	))

	// mahua gallery
	galleryActivateAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("麻花最萌啦"))
		context.SetData([]string{})
		return
	})

	galleryInactiveAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("谢谢爸爸"))
		messageIDs := context.GetData().([]string)
		for _, id := range messageIDs {
			if originURL, thumbnailURL, err := fetchAndUploadContent(id, "mahua"); err == nil {
				messages = append(messages, linebot.NewTextMessage(id+"\n\n"+originURL+"\n"+thumbnailURL))
			} else {
				log.Println(err)
			}
		}
		return
	})

	galleryUsualAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messageIDs := context.GetData().([]string)
		switch message := event.Message.(type) {
		case *linebot.ImageMessage:
			messageIDs = append(messageIDs, message.ID)
		case *linebot.TextMessage:
			switch message.Text {
			case "u":
				if len(messageIDs) == 0 {
					break
				}
				messageIDs = messageIDs[:len(messageIDs)-1]
			}
		}
		messages = append(messages, linebot.NewTextMessage("照照：\n"+strings.Join(messageIDs, "\n")))
		context.SetData(messageIDs)
		return
	})

	dispatcher.RegisterWithID([]string{"mg"}, []string{laosiji}, actionDispatcher.NewContextAction(
		[]string{"s"},
		galleryActivateAction,
		galleryInactiveAction,
		galleryUsualAction,
	))

	defaultAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		switch event.Source.Type {
		case linebot.EventSourceTypeUser:
			messages = forwardToTuling(event, messages)
		default:
			if message, ok := event.Message.(*linebot.TextMessage); ok {
				if !(strings.HasPrefix(message.Text, "@mahua") || strings.HasPrefix(message.Text, "@mh") || strings.HasPrefix(message.Text, "@麻花")) {
					return
				}
				replacer := strings.NewReplacer("@mh", "", "@mahua", "", "@麻花", "")
				message.Text = replacer.Replace(message.Text)
				messages = forwardToTuling(event, messages)
			}
		}
		return
	})
	dispatcher.RegisterDefaultAction(defaultAction)
}
