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

	"github.com/bamwang/mahua-bot/action_dispatcher"
	"github.com/line/line-bot-sdk-go/linebot"
)

var moyu = os.Getenv("MOYU_ID")
var laosiji = os.Getenv("LAOSIJI_ID")
var address = os.Getenv("LAOSIJI_ADD")

func register(dispatcher *actionDispatcher.ActionDispatcher, collections map[string]*mgo.Collection) {

	subscribers, publications, exercises, exercisesMeta, groups :=
		collections["subscribers"], collections["publications"], collections["exercises"], collections["exercisesMeta"], collections["groups"]

	// f23
	f23MenuHandler := func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		donburiCol := linebot.NewCarouselColumn(fmt.Sprintf("%s/don.jpg", staticsPrefix), "Donburi / Curry", "JPY300", linebot.NewURITemplateAction("BUY", "https://webpos.line.me/cafe/payment/c81e728d9d4c2f636f067f89cc14862c/reserve"))
		saladCol := linebot.NewCarouselColumn(fmt.Sprintf("%s/salad.jpg", staticsPrefix), "Salad", "JPY100", linebot.NewURITemplateAction("BUY", "https://webpos.line.me/cafe/payment/a87ff679a2f3e71d9181a67b7542122c/reserve"))
		bentoCol := linebot.NewCarouselColumn(fmt.Sprintf("%s/bento.jpg", staticsPrefix), "Bento", "JPY500", linebot.NewURITemplateAction("BUY", "https://webpos.line.me/cafe/payment/c4ca4238a0b923820dcc509a6f75849b/reserve"))
		soupCol := linebot.NewCarouselColumn(fmt.Sprintf("%s/soup.jpg", staticsPrefix), "Soup", "JPY300", linebot.NewURITemplateAction("BUY", "https://webpos.line.me/cafe/payment/eccbc87e4b5ce2fe28308fd9f2a7baf3/reserve"))
		messages = append(messages, linebot.NewTemplateMessage("23F cafe menu", linebot.NewCarouselTemplate(donburiCol, saladCol, bentoCol, soupCol)))
		return
	}
	dispatcher.RegisterWithType([]string{"23b"}, []linebot.EventSourceType{}, "显示食堂购买支付菜单", actionDispatcher.NewReplayAction(f23MenuHandler))

	// msg
	// MsgHandler := func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
	// 	message := createMassageMessage(event.Source.GroupID+event.Source.RoomID, massages)
	// 	messages = append(messages, message)
	// 	return
	// }
	// dispatcher.RegisterWithType([]string{"msg"}, []linebot.EventSourceType{}, actionDispatcher.NewReplayAction(MsgHandler))

	// msgs
	MsgsHandler := func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		id := actionDispatcher.ExtractTargetID(event)
		sendTo([]string{id}, "正在查询中")
		messages = forwardToMsgc(event, messages)

		return
	}
	dispatcher.RegisterWithType([]string{"msgs"}, []linebot.EventSourceType{}, "查询当前空余马杀鸡情况", actionDispatcher.NewReplayAction(MsgsHandler))

	// mahua gallery
	MGHandler := func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		mahuas := loadMahua()
		if len(mahuas) == 0 {
			return
		}
		rand.Seed(time.Now().UnixNano())
		mahuaOrigin := mahuas[rand.Intn(len(mahuas))]
		mahuaBase := mahuaOrigin[:len(mahuaOrigin)-4]
		messages = append(messages, linebot.NewImageMessage(bucketURLBase+mahuaOrigin, bucketURLBase+mahuaBase+"_thumbnail.jpg"))
		return
	}
	dispatcher.RegisterWithType([]string{"看麻花"}, []linebot.EventSourceType{}, "随机看一张麻花的照片", actionDispatcher.NewReplayAction(MGHandler))

	// mahua gallery
	MGNHandler := func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		mahuas := loadMahua()
		if len(mahuas) == 0 {
			return
		}
		mahuaOrigin := mahuas[len(mahuas)-1]
		mahuaBase := mahuaOrigin[:len(mahuaOrigin)-4]
		messages = append(messages, linebot.NewImageMessage(bucketURLBase+mahuaOrigin, bucketURLBase+mahuaBase+"_thumbnail.jpg"))
		return
	}
	dispatcher.RegisterWithType([]string{"最新麻花"}, []linebot.EventSourceType{}, "看最新的麻花照片", actionDispatcher.NewReplayAction(MGNHandler))

	// mahua gallery subscription
	MGSHandler := func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		name := getUserName(event.Source.UserID)
		id := actionDispatcher.ExtractTargetID(event)
		has, err := checkSubscription(id, "mg", subscribers)
		if err != nil {
			return
		}
		if has {
			messages = append(messages, linebot.NewTextMessage("已经在这订阅过麻花啦！"))
			return
		}
		err = subscribe(id, "mg", name, event.Source.Type, subscribers)
		if err != nil {
			return
		}
		messages = append(messages, linebot.NewTextMessage("麻花有新照照的时候都会第一时间通知你哒！"))
		sendTo([]string{laosiji}, fmt.Sprintf("%s (%v) 订阅了麻花", name, event.Source.Type))
		return
	}
	dispatcher.RegisterWithType([]string{"订阅麻花"}, []linebot.EventSourceType{}, "当有最新的麻花时通知你", actionDispatcher.NewReplayAction(MGSHandler))

	// mahua gallery unsubscription
	MGSCHandler := func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		name := getUserName(event.Source.UserID)
		id := actionDispatcher.ExtractTargetID(event)
		has, err := checkSubscription(id, "mg", subscribers)
		if err != nil {
			return
		}
		if !has {
			messages = append(messages, linebot.NewTextMessage("哼！你本来就没订阅麻花"))
			return
		}
		err = unsubscribe(id, "mg", event.Source.Type, subscribers)
		if err != nil {
			return
		}
		messages = append(messages, linebot.NewTextMessage("你不喜欢麻花了吗？呜呜~~"))
		sendTo([]string{laosiji}, fmt.Sprintf("%s (%v) 退订了麻花", name, event.Source.Type))
		return
	}
	dispatcher.RegisterWithType([]string{"退订麻花"}, []linebot.EventSourceType{}, "取消订阅", actionDispatcher.NewReplayAction(MGSCHandler))

	// mahua gallery unsubscription
	MGPCHandler := func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
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
	dispatcher.RegisterWithID([]string{"cp"}, []string{laosiji}, "取消之后的群发", actionDispatcher.NewReplayAction(MGPCHandler))

	// fan
	fanActivateAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		name := getUserName(event.Source.UserID)

		userMap := map[string]string{
			event.Source.UserID: name,
		}
		messages = append(messages, linebot.NewTextMessage("好的, "+name))
		context.SetData(userMap)
		return
	})

	fanInactiveAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("麻花也去觅食啦！喵~~"))
		return
	})

	fanUsualAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
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
				prefix := ""
				weekday := time.Now().Weekday()
				if weekday == time.Monday || weekday == time.Thursday {
					prefix = "今天是吃刀削面的日子（" + weekday.String() + ")\n"
				}
				messages = append(messages, linebot.NewTextMessage(prefix+"走啦走啦：\n"+strings.Join(names, "\n")))
			}
		}
		return
	})

	dispatcher.RegisterWithType([]string{"f"}, []linebot.EventSourceType{linebot.EventSourceTypeGroup, linebot.EventSourceTypeRoom}, "报告你要去参加群饭; nf 取消; c 查询现在状态; g 提醒大家出发; g! 停止募集", actionDispatcher.NewContextAction(
		[]string{"g!"},
		fanActivateAction,
		fanInactiveAction,
		fanUsualAction,
	))

	// mahua
	mahuaActivateAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("喵~~"))
		return
	})

	mahuaInactiveAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("汪~~"))
		return
	})

	mahuaUsualAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = forwardToTuling(event, messages)
		return
	})

	dispatcher.RegisterWithType([]string{"麻花来"}, []linebot.EventSourceType{linebot.EventSourceTypeGroup, linebot.EventSourceTypeRoom}, "让麻花参与聊天; 麻花拜拜: 让麻花离开", actionDispatcher.NewContextAction(
		[]string{"麻花拜拜"},
		mahuaActivateAction,
		mahuaInactiveAction,
		mahuaUsualAction,
	))

	// futi
	futiActivateAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("麻花为你代言"))
		return
	})

	futiInactiveAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("麻花还是麻花"))
		return
	})

	futiUsualAction := actionDispatcher.NewPushAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, id string, err error) {
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

	dispatcher.RegisterWithType([]string{"ft"}, []linebot.EventSourceType{linebot.EventSourceTypeUser}, "以麻花身份在水群里发言; lt: 停止附体行为", actionDispatcher.NewContextAction(
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

	galleryActivateAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		messages = append(messages, linebot.NewTextMessage("麻花最萌啦"))
		context.SetData(gallery{[]string{}, false})
		return
	})

	galleryInactiveAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
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

	galleryUsualAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
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

	dispatcher.RegisterWithID([]string{"mg"}, []string{laosiji}, "传麻花照片", actionDispatcher.NewContextAction(
		[]string{"s"},
		galleryActivateAction,
		galleryInactiveAction,
		galleryUsualAction,
	))

	dispatcher.RegisterWithID([]string{"hash"}, []string{laosiji}, "显示eth挖矿状态", actionDispatcher.NewReplayAction(
		func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
			messages = getMiningStatus(event, messages, address, "")
			return messages, err
		},
	))

	em := exercisesManager{exercisesMeta, exercises}

	dispatcher.RegisterWithID([]string{"js+"}, []string{moyu}, "健身+1", actionDispatcher.NewReplayAction(
		func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
			message, err := em.add(event.Source.UserID)
			messages = append(messages, linebot.NewTextMessage(message))
			return messages, err
		},
	))

	dispatcher.RegisterWithID([]string{"js-"}, []string{moyu}, "健身-1", actionDispatcher.NewReplayAction(
		func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
			message, err := em.remove(event.Source.UserID)
			messages = append(messages, linebot.NewTextMessage(message))
			return messages, err
		},
	))

	dispatcher.RegisterWithID([]string{"jsc"}, []string{moyu}, "健身查询", actionDispatcher.NewReplayAction(
		func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
			message, err := em.check("", "")
			messages = append(messages, linebot.NewTextMessage(message))
			return messages, err
		},
	))

	dispatcher.RegisterWithID([]string{"jsc.m"}, []string{moyu}, "本月健身查询", actionDispatcher.NewReplayAction(
		func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
			message, err := em.check("m", "")
			messages = append(messages, linebot.NewTextMessage(message))
			return messages, err
		},
	))

	dispatcher.RegisterWithID([]string{"jsc.w"}, []string{moyu}, "本周健身查询", actionDispatcher.NewReplayAction(
		func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
			message, err := em.check("w", "")
			messages = append(messages, linebot.NewTextMessage(message))
			return messages, err
		},
	))

	dispatcher.RegisterWithID([]string{"jsj"}, []string{moyu}, "参与健身计划", actionDispatcher.NewReplayAction(
		func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
			message, err := em.join(event.Source.UserID)
			messages = append(messages, linebot.NewTextMessage(message))
			return messages, err
		},
	))

	dispatcher.RegisterWithType([]string{"dk"}, []linebot.EventSourceType{linebot.EventSourceTypeUser}, "打卡: 输入dk取得详细说明", actionDispatcher.NewReplayAction(
		func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
			messages = forwardToE4628(event, messages)
			return messages, err
		},
	))

	dispatcher.RegisterWithType([]string{"qr"}, []linebot.EventSourceType{linebot.EventSourceTypeUser}, "忘带社员证时发行QR: 输入qr取得详细说明", actionDispatcher.NewReplayAction(
		func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
			messages = forwardToQR(event, messages)
			return messages, err
		},
	))

	dispatcher.RegisterWithID([]string{"clova"}, []string{moyu, laosiji}, "clova 日语文本: 调戏clova", actionDispatcher.NewReplayAction(
		func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
			messages = forwardToClova(event, messages)
			return messages, err
		},
	))

	msgSubHandlder := func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		name := getUserName(event.Source.UserID)
		id := actionDispatcher.ExtractTargetID(event)
		has, err := checkSubscription(id, "msg", subscribers)
		if err != nil {
			return
		}
		if has {
			messages = append(messages, linebot.NewTextMessage("已经在这订阅过马杀鸡通知了！"))
			return
		}
		err = subscribe(id, "msg", name, event.Source.Type, subscribers)
		if err != nil {
			return
		}
		messages = append(messages, linebot.NewTextMessage("我会第一时间通知你空余的马杀鸡哒！"))
		return
	}
	dispatcher.RegisterWithType([]string{"msgsub"}, []linebot.EventSourceType{}, "10~19点的每个整点05分，通知你前的空余马杀鸡", actionDispatcher.NewReplayAction(msgSubHandlder))

	// mahua gallery unsubscription
	msgUnsubHandler := func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		id := actionDispatcher.ExtractTargetID(event)
		has, err := checkSubscription(id, "msg", subscribers)
		if err != nil {
			return
		}
		if !has {
			messages = append(messages, linebot.NewTextMessage("已并未订阅通知"))
			return
		}
		err = unsubscribe(id, "msg", event.Source.Type, subscribers)
		if err != nil {
			return
		}
		messages = append(messages, linebot.NewTextMessage("已退订通知"))
		return
	}
	dispatcher.RegisterWithType([]string{"msgunsub"}, []linebot.EventSourceType{}, "取消订阅马杀鸡", actionDispatcher.NewReplayAction(msgUnsubHandler))

	defaultAction := actionDispatcher.NewReplayAction(func(event *linebot.Event, context *actionDispatcher.Context) (messages []linebot.Message, err error) {
		switch event.Source.Type {
		case linebot.EventSourceTypeUser:

			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				return forwardToTuling(event, messages), nil

			case *linebot.ImageMessage:
				return decodeQR(message, messages)
			}

		default:
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				if strings.HasPrefix(message.Text, "@mahua") || strings.HasPrefix(message.Text, "@mh") || strings.HasPrefix(message.Text, "@麻花") {
					replacer := strings.NewReplacer("@mh", "", "@mahua", "", "@麻花", "")
					message.Text = replacer.Replace(message.Text)
					return forwardToTuling(event, messages), nil
				}
				if strings.HasPrefix(message.Text, "@all") || strings.HasPrefix(message.Text, "@here") || strings.HasPrefix(message.Text, "@所有人") {
					replacer := strings.NewReplacer("@all", "", "@here", "", "@所有人", "")
					message.Text = replacer.Replace(message.Text)
					// if event.Source.Type == linebot.EventSourceTypeRoom {
					// 	res, _err := bot.GetRoomMemberIDs(event.Source.RoomID, os.Getenv("CHANNEL_ACCSESS_TOKEN")).Do()
					// 	if _err != nil {
					// 		err = _err
					// 		return
					// 	}
					// 	userIDs = res.MemberIDs
					// }
					// if event.Source.Type == linebot.EventSourceTypeGroup {
					// 	res, _err := bot.GetGroupMemberIDs(event.Source.GroupID, os.Getenv("CHANNEL_ACCSESS_TOKEN")).Do()
					// 	if _err != nil {
					// 		err = _err
					// 		return
					// 	}
					// 	userIDs = res.MemberIDs
					// }
					var group Group
					groupID := actionDispatcher.ExtractTargetID(event)
					err := groups.FindId(groupID).One(&group)
					if err != nil {
						return nil, err
					}
					ids := make([]string, 0, len(group.Users))
					for id := range group.Users {
						ids = append(ids, id)
					}
					if groupID == moyu {
						message.Text = "来自摸鱼的消息：\n" + message.Text
					}
					sendTo(ids, message.Text)
				}
			case *linebot.ImageMessage:
				return decodeQR(message, messages)
			}
		}
		return
	})
	dispatcher.RegisterDefaultAction(defaultAction, "")
}
