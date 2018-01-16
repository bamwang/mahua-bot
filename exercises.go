package main

import (
	"fmt"
	"sort"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type meta struct {
	UserID    string    `bson:"userID"`
	CreatedAt time.Time `bson:"createdAt"`
}

type exercises struct {
	ID        bson.ObjectId `bson:"_id"`
	UserID    string        `bson:"userID"`
	CreatedAt time.Time     `bson:"createdAt"`
	IsDeleted bool          `bson:"_isDeleted"`
}

type exercisesManager struct {
	exercisesMeta *mgo.Collection
	exercises     *mgo.Collection
}

func (e *exercisesManager) join(userID string) (message string, err error) {
	has, err := e.has(userID)
	if err != nil {
		return
	}
	if has {
		message = "你已经加入了！"
		return
	}
	return "欢迎加入下班不划水健身俱乐部！", e.exercisesMeta.Insert(meta{userID, time.Now()})
}

func (e *exercisesManager) has(userID string) (has bool, err error) {
	n, err := e.exercisesMeta.Find(bson.M{
		"userID": userID,
	}).Count()
	if err != nil {
		return
	}
	has = n > 0
	return
}

func (e *exercisesManager) add(userID string) (message string, err error) {
	has, err := e.has(userID)
	if err != nil {
		return
	}
	if !has {
		message = "请先发送jsj加入下班后不划水健身俱乐部\n一旦加入就不能退会哦"
		return
	}
	err = e.exercises.Insert(exercises{bson.NewObjectId(), userID, time.Now(), false})
	if err != nil {
		return
	}
	return e.check("", "")
}

func bod(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func (e *exercisesManager) remove(userID string) (message string, err error) {
	has, err := e.has(userID)
	if err != nil {
		return
	}
	if !has {
		message = "请先发送jsj加入下班后不划水健身俱乐部"
		return
	}
	var ex exercises
	err = e.exercises.Find(bson.M{
		"userID":     userID,
		"_isDeleted": false,
		"createdAt":  bson.M{"$gte": bod(time.Now())},
	}).One(&ex)
	if err != nil {
		return
	}
	if ex.ID == "" {
		message = "嗯哼"
		return
	}
	ex.IsDeleted = true
	err = e.exercises.UpdateId(ex.ID, ex)
	if err != nil {
		return
	}
	return e.check("", "好吧")
}

type Entry struct {
	userID string
	count  int
}
type List []Entry

func (l List) Len() int {
	return len(l)
}

func (l List) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l List) Less(i, j int) bool {
	return (l[i].count < l[j].count)
}

func (e *exercisesManager) check(flag, prefix string) (message string, err error) {
	year, month, _ := time.Now().Date()
	var start time.Time
	switch flag {
	case "w":
		fallthrough
	case "m":
		start = time.Date(year, month, 0, 0, 0, 0, 0, time.Now().Location())
	default:
		start = time.Date(year, 0, 0, 0, 0, 0, 0, time.Now().Location())
	}
	var exs []exercises

	err = e.exercises.Find(bson.M{
		"_isDeleted": false,
		"createdAt":  bson.M{"$gte": start},
	}).Sort("createdAt").All(&exs)
	if err != nil {
		return
	}
	message += prefix + "\n"
	if len(exs) == 0 {
		message = "还木有人健身"
		return
	}
	rankMap := map[string]int{}
	for _, ex := range exs {
		message += ex.CreatedAt.Format("2006-01-02  ") + getUserName(ex.UserID) + "\n"
		rankMap[ex.UserID]++
	}

	rank := List{}
	for k, v := range rankMap {
		e := Entry{k, v}
		rank = append(rank, e)
	}
	sort.Sort(rank)
	message += "===========\n"
	for _, ent := range rank {
		message += fmt.Sprintf("%s : %d", getUserName(ent.userID), ent.count) + "\n"
	}
	return
}
