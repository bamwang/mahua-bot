package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"bitbucket.com/wangzhucn/mahua-bot/action_dispatcher"
	"github.com/line/line-bot-sdk-go/linebot"
)

var moyu = os.Getenv("MOYU_ID")
var laosiji = os.Getenv("LAOSIJI_ID")

func register(dispatcher *actionDispatcher.ActionDispatcher, massages, subscribers, publications *mgo.Collection) {

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

	// mahua gallery subscription
	MGSHandler := func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		name := getUserName(event.Source.UserID)
		var id string
		switch event.Source.Type {
		case linebot.EventSourceTypeUser:
			id = event.Source.UserID
		case linebot.EventSourceTypeRoom:
			id = event.Source.RoomID
		case linebot.EventSourceTypeGroup:
			id = event.Source.GroupID
		}
		n, _ := subscribers.Find(bson.M{
			"uid": id,
		}).Count()
		if n == 1 {
			messages = append(messages, linebot.NewTextMessage("已经在这订阅过麻花啦！"))
			return
		}
		subscribers.Insert(bson.M{
			"uid":          id,
			"name":         name, // will not be upadated automatically
			"type":         event.Source.Type,
			"subscribedAt": bson.Now(),
		})
		messages = append(messages, linebot.NewTextMessage("麻花有新照照的时候都会第一时间通知你哒！"))
		sendTo([]string{laosiji}, fmt.Sprintf("%s (%v) 订阅了麻花", name, event.Source.Type))
		return
	}
	dispatcher.RegisterWithType([]string{"订阅麻花"}, []linebot.EventSourceType{}, actionDispatcher.NewReplayAction(MGSHandler))

	// mahua gallery unsubscription
	MGSCHandler := func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		name := getUserName(event.Source.UserID)
		var id string
		switch event.Source.Type {
		case linebot.EventSourceTypeUser:
			id = event.Source.UserID
		case linebot.EventSourceTypeRoom:
			id = event.Source.RoomID
		case linebot.EventSourceTypeGroup:
			id = event.Source.GroupID
		}
		n, _ := subscribers.Find(bson.M{
			"uid": id,
		}).Count()
		if n == 0 {
			messages = append(messages, linebot.NewTextMessage("哼！你本来就没订阅麻花"))
			return
		}
		subscribers.RemoveAll(bson.M{
			"uid": id,
		})
		messages = append(messages, linebot.NewTextMessage("你不喜欢麻花了吗？呜呜~~"))
		sendTo([]string{laosiji}, fmt.Sprintf("%s (%v) 退订了麻花", name, event.Source.Type))
		return
	}
	dispatcher.RegisterWithType([]string{"退订麻花"}, []linebot.EventSourceType{}, actionDispatcher.NewReplayAction(MGSCHandler))

	// mahua gallery unsubscription
	MGPCHandler := func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		pub := publication{}
		publications.Find(nil).Sort("-_id").One(&pub)
		if pub.PublishedAt.Year() >= 2017 {
			messages = append(messages, linebot.NewTextMessage("无群发"))
			return
		}
		pub.PublishedAt = time.Date(2100, 1, 1, 0, 0, 0, 0, time.Local)
		publications.UpdateId(pub.ID, pub)
		messages = append(messages, linebot.NewTextMessage("停止群发麻花"))
		return
	}
	dispatcher.RegisterWithID([]string{"cp"}, []string{laosiji}, actionDispatcher.NewReplayAction(MGPCHandler))

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

				sendTo(userIDs, name+"喊道：走啦走啦！去吃饭啦！")
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
	type gallery struct {
		ids           []string
		shouldPublish bool
	}

	galleryActivateAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("麻花最萌啦"))
		context.SetData(gallery{[]string{}, false})
		return
	})

	galleryInactiveAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("谢谢爸爸"))
		galleryObj := context.GetData().(gallery)
		urls := ""
		ids := []string{}
		for _, id := range galleryObj.ids {
			if originURL, thumbnailURL, err := fetchAndUploadContent(id, "mahua"); err == nil {
				urls += fmt.Sprintf("%s\n\n%s\n%s\n", id, originURL, thumbnailURL)
				ids = append(ids, id)
			} else {
				log.Println(err)
			}
		}
		if galleryObj.shouldPublish {
			publications.Insert(publication{
				IDs:       ids,
				CreatedAt: bson.Now(),
			})
		}
		messages = append(messages, linebot.NewTextMessage(urls))
		return
	})

	galleryUsualAction := actionDispatcher.NewReplayAction(func(event linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		galleryObj := context.GetData().(gallery)
		switch message := event.Message.(type) {
		case *linebot.ImageMessage:
			galleryObj.ids = append(galleryObj.ids, message.ID)
		case *linebot.TextMessage:
			switch message.Text {
			case "u":
				if len(galleryObj.ids) == 0 {
					break
				}
				galleryObj.ids = galleryObj.ids[:len(galleryObj.ids)-1]
			case "p":
				galleryObj.shouldPublish = !galleryObj.shouldPublish
			}
		}
		messages = append(messages, linebot.NewTextMessage(
			fmt.Sprintf("照照：\n%s\n	是否发布: %t", strings.Join(galleryObj.ids, "\n"), galleryObj.shouldPublish)),
		)
		context.SetData(galleryObj)
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
