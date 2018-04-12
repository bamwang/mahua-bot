package main

import (
	"log"

	"github.com/line/line-bot-sdk-go/linebot"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func subscribe(id, key, name string, sourceType linebot.EventSourceType, subscribers *mgo.Collection) (err error) {
	_, err = subscribers.Upsert(bson.M{
		"uid": id,
	}, bson.M{
		"$set": bson.M{
			"name":         name, // will not be upadated automatically
			"type":         sourceType,
			"updatedAt":    bson.Now(),
			"items." + key: true,
		},
	})
	return err
}

func checkSubscription(id, key string, subscribers *mgo.Collection) (has bool, err error) {
	n, err := subscribers.Find(bson.M{
		"uid":          id,
		"itmes." + key: true,
	}).Count()
	has = n > 0
	return
}

func unsubscribe(id, key string, sourceType linebot.EventSourceType, subscribers *mgo.Collection) (err error) {
	return subscribers.Update(bson.M{
		"uid": id,
	}, bson.M{
		"$set": bson.M{
			"itmes." + key: false,
		},
	})
}

func getSubscriberIDs(key string, subscribers *mgo.Collection) (ids []string, err error) {
	var subs []map[string]string
	err = subscribers.Find(bson.M{
		"itmes." + key: true,
	}).All(&subs)
	log.Println(subs)
	if err != nil {
		return
	}
	for _, subscriber := range subs {
		ids = append(ids, subscriber["uid"])
	}
	return
}
