package main

import (
	"crypto/tls"
	"encoding/json"
	"image"
	"image/jpeg"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"strconv"

	"github.com/bamwang/mahua-bot/action_dispatcher"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/nfnt/resize"
)

var (
	host           = os.Getenv("HOST")
	staticsPrefix  = host + "/statics"
	contentsPrefix = host + "/contents"
)

var idMap = map[string]string{}

var nameMap = map[string]string{}

var bot *linebot.Client

type Block struct {
	Date      string   `json:"date"`
	Time      string   `json:"time"`
	Massagers []string `json:"massagers"`
}
type Blocks struct {
	Blocks []Block `json:"blocks"`
}

type SendRequest struct {
	ID   string `json:id`
	Text string `json:text`
}

type publication struct {
	ID          bson.ObjectId `bson:"_id,omitempty"`
	IDs         []string      `bson:"ids"`
	CreatedAt   time.Time     `bson:"createdAt,omitempty"`
	PublishedAt time.Time     `bson:"publishedAt,omitempty"`
}

func init() {
	var err error
	bot, err = linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_ACCSESS_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}
	initS3()
}

func main() {
	log.Println("start")
	url := os.Getenv("MONGODB_URI")
	tlsConfig := &tls.Config{}
	info, err := mgo.ParseURL(url)
	if err != nil {
		log.Panicln(err)
	}
	info.Timeout = time.Second * 10
	info.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
		conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
		return conn, err
	}
	session, err := mgo.DialWithInfo(info)
	if err != nil {
		log.Panicln(err)
	}
	defer session.Close()

	dispatcher := actionDispatcher.New(bot)

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	massages := session.DB("").C("massages")
	subscribers := session.DB("").C("subscribers")
	publications := session.DB("").C("publications")
	exercises := session.DB("").C("exercises")
	exercisesMeta := session.DB("").C("exercisesMeta")

	register(&dispatcher, massages, subscribers, publications, exercises, exercisesMeta)

	fs := http.FileServer(http.Dir("statics"))
	fsContent := http.FileServer(http.Dir("contents"))
	http.Handle("/statics/", http.StripPrefix("/statics/", fs))
	http.Handle("/contents/", http.StripPrefix("/contents/", fsContent))

	http.HandleFunc("/msg", func(w http.ResponseWriter, req *http.Request) {

		blocks := Blocks{}
		var err error
		w.Header().Add("Access-Control-Allow-Origin", "*")
		if req.Method == http.MethodGet {
			messages := forwardToMsgc(nil, []linebot.Message{})
			if _, err := bot.PushMessage(moyu, messages[0]).Do(); err != nil {
				w.WriteHeader(500)
				w.Write([]byte("fail"))
				log.Println(err)
				return
			}
			w.Write([]byte("done"))
			return
		}
		if req.Method == http.MethodPost {
			err = json.NewDecoder(req.Body).Decode(&blocks)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			err = massages.Insert(blocks)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = massages.Find(nil).Sort("-_id").One(&blocks)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(405)
		return
	})

	http.HandleFunc("/send", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}

		var sendReq SendRequest
		err := json.NewDecoder(req.Body).Decode(&sendReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var id string
		var has bool
		if id, has = idMap[sendReq.ID]; !has {
			id = sendReq.ID
		}
		if _, err := bot.PushMessage(id, linebot.NewTextMessage(sendReq.Text)).Do(); err != nil {
			w.WriteHeader(500)
			w.Write([]byte("fail"))
			log.Println(err)
			return
		}
		w.Write([]byte("done"))
	})

	http.HandleFunc("/publish", func(w http.ResponseWriter, req *http.Request) {

		subs := []map[string]string{}
		subscribers.Find(nil).All(&subs)
		ids := []string{}
		for _, subscriber := range subs {
			ids = append(ids, subscriber["uid"])
		}
		publish(w, req, publications, ids, "最新麻花来啦！", true)
	})

	http.HandleFunc("/publish_test", func(w http.ResponseWriter, req *http.Request) {
		publish(w, req, publications, []string{laosiji}, "最新麻花群发测试", false)
	})

	http.HandleFunc("/noti", func(w http.ResponseWriter, req *http.Request) {
		data := map[string]string{}
		err = json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("%+v", data)
		id := data["id"]
		message := data["message"]
		if id == "" || message == "" {
			w.WriteHeader(400)
			return
		}
		sendTo([]string{id}, message)
	})

	// Setup HTTP Server for receiving requests from LINE platform
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}
		for _, event := range events {
			b, _ := json.Marshal(event)
			log.Println("REQ: " + string(b))
			dispatcher.Dispatch(&event)
		}
	})

	// This is just sample code.
	// For actual use, you must support HTTPS by using `ListenAndServeTLS`, a reverse proxy or something else.
	var port string
	if port = os.Getenv("PORT"); port == "" {
		port = strconv.Itoa(3000)
	}
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func publish(w http.ResponseWriter, req *http.Request, publications *mgo.Collection, ids []string, message string, updateStatus bool) {
	log.Println(ids)

	if req.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}

	pub := publication{}
	publications.Find(nil).Sort("-_id").One(&pub)
	messages := []linebot.Message{linebot.NewTextMessage(message)}
	for _, id := range pub.IDs {
		messages = append(messages, linebot.NewImageMessage(bucketURLPrefix+"mahua/"+id+".jpg", bucketURLPrefix+"mahua/"+id+"_thumbnail.jpg"))
	}
	if pub.PublishedAt.Year() >= 2017 {
		w.WriteHeader(404)
		return
	}

	var err error
	for _, id := range ids {
		_, err = bot.PushMessage(id, messages...).Do()
	}
	if err == nil {
		w.WriteHeader(200)
		if updateStatus {
			pub.PublishedAt = bson.Now()
			publications.UpdateId(pub.ID, pub)
		}
		return
	}
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
}

func getUserName(userID string) string {
	name := "没加麻花为好友的路人"
	if resp, err := bot.GetProfile(userID).Do(); err == nil {
		name = resp.DisplayName
	}
	return name
}

func sendTo(ids []string, messageText string) {
	for _, id := range ids {
		bot.PushMessage(id, linebot.NewTextMessage(messageText)).Do()
	}
}

func fetchAndUploadContent(messageID string, dir string) (string, string, error) {
	res, err := bot.GetMessageContent(messageID).Do()
	if err != nil {
		return "", "", err
	}

	img, _, err := image.Decode(res.Content)
	thumbnail := resize.Resize(240, 0, img, resize.Lanczos3)

	// Create the file
	out, err := os.Create("/tmp/" + messageID + ".jpg")
	if err != nil {
		return "", "", err
	}
	defer out.Close()

	// Writer the body to file
	err = jpeg.Encode(out, img, nil)
	if err != nil {
		return "", "", err
	}

	// Create the file
	outThumb, err := os.Create("/tmp/" + messageID + "_thumbnail.jpg")
	if err != nil {
		return "", "", err
	}
	defer outThumb.Close()

	// Writer the body to file
	err = jpeg.Encode(outThumb, thumbnail, &jpeg.Options{Quality: 80})
	if err != nil {
		return "", "", err
	}
	url, err := upload(messageID+".jpg", "/tmp/", dir, true)
	if err != nil {
		return "", "", err
	}
	thumbnailURL, err := upload(messageID+"_thumbnail.jpg", "/tmp/", dir, true)
	return url, thumbnailURL, err
}
