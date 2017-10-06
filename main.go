package main

import (
	"encoding/json"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"

	mgo "gopkg.in/mgo.v2"

	"strconv"

	"bitbucket.com/wangzhucn/mahua-bot/action_dispatcher"

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
	url := os.Getenv("MONGODB_URI")
	session, err := mgo.Dial(url)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	dispatcher := actionDispatcher.New(bot)

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	massages := session.DB("").C("massages")

	register(&dispatcher, massages)

	fs := http.FileServer(http.Dir("statics"))
	fsContent := http.FileServer(http.Dir("contents"))
	http.Handle("/statics/", http.StripPrefix("/statics/", fs))
	http.Handle("/contents/", http.StripPrefix("/contents/", fsContent))

	http.HandleFunc("/msg", func(w http.ResponseWriter, req *http.Request) {

		blocks := Blocks{}
		var err error
		w.Header().Add("Access-Control-Allow-Origin", "*")
		if req.Method == http.MethodGet {
			err = massages.Find(nil).Sort("-_id").One(&blocks)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			b, err := json.Marshal(blocks)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if _, err := bot.PushMessage(moyu, createMassageInfo(massages)).Do(); err != nil {
				w.WriteHeader(500)
				w.Write([]byte("fail"))
				log.Println(err)
				return
			}
			w.Write(b)
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
		w.WriteHeader(200)
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
			b, _ := event.MarshalJSON()
			log.Println("REQ: " + string(b))
			dispatcher.Dispatch(*event)
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

func getUserName(userID string) string {
	name := "没加麻花为好友的路人"
	if resp, err := bot.GetProfile(userID).Do(); err == nil {
		name = resp.DisplayName
	}
	return name
}

func sentTo(ids []string, messageText string) {
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
