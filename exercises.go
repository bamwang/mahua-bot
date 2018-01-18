package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/jinzhu/now"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type meta struct {
	UserID    string    `bson:"userID"`
	CreatedAt time.Time `bson:"createdAt"`
}

type exercises struct {
	ID        bson.ObjectId `bson:"_id,omitempty"`
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
	if count, _err := e.exercises.Find(bson.M{"userID": userID, "_isDeleted": false, "createdAt": bson.M{"$gte": now.BeginningOfDay()}}).Count(); count > 0 {
		if _err != nil {
			err = _err
			return
		}
		message = "一天只能打卡一次哦"
		return
	}
	err = e.exercises.Insert(exercises{bson.NewObjectId(), userID, time.Now(), false})
	if err != nil {
		return
	}
	return e.check("w", "偉い！偉い！偉い！偉い！偉い！git add ")
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
		"createdAt":  bson.M{"$gte": now.BeginningOfDay()},
	}).One(&ex)
	if err == mgo.ErrNotFound {
		return "你没说你要健身来着", nil
	}
	if err != nil {
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
	return (l[i].count > l[j].count)
}

func (e *exercisesManager) check(flag, prefix string) (message string, err error) {
	var start time.Time
	var title string
	now.FirstDayMonday = true
	switch flag {
	case "w":
		title = "本周统计"
		start = now.BeginningOfWeek()
	case "m":
		title = "本月统计"
		start = now.BeginningOfMonth()
	default:
		title = "全年统计"
		start = now.BeginningOfYear()
	}
	var exs []exercises

	err = e.exercises.Find(bson.M{
		"_isDeleted": false,
		"createdAt":  bson.M{"$gte": start},
	}).Sort("createdAt").All(&exs)
	if err != nil {
		return
	}
	if prefix != "" {
		message += prefix + "\n"
	}

	message += "\n===" + title + "===\n"
	if len(exs) == 0 {
		message += "还木有人健身"
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
	message += "====排行榜====\n"
	for _, ent := range rank {
		message += fmt.Sprintf("%s : %d", getUserName(ent.userID), ent.count) + "\n"
	}
	return
}
