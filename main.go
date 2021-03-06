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
	"os/exec"
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

type User struct {
	ID           string `bson:"_id" json:"id"`
	Name         string `bson:"name" json:"name"`
	AccessToken  string `bson:"accessToken" json:"accessToken"`
	PortraitURL  string `bson:"portraitURL" json:"portraitURL"`
	RefreshToken string `bson:"refreshToken" json:"refreshToken"`
}

type Group struct {
	ID    string                  `bson:"_id" json:"id"`
	Type  linebot.EventSourceType `bson:"type" json:"type"`
	Users map[string]interface{}  `bson:"users" json:"users"`
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

	users := session.DB("").C("users")
	massages := session.DB("").C("massages")
	subscribers := session.DB("").C("subscribers")
	publications := session.DB("").C("publications")
	exercises := session.DB("").C("exercises")
	exercisesMeta := session.DB("").C("exercisesMeta")
	groups := session.DB("").C("groups")

	collections := map[string]*mgo.Collection{
		"users":         users,
		"massages":      massages,
		"subscribers":   subscribers,
		"publications":  publications,
		"exercises":     exercises,
		"exercisesMeta": exercisesMeta,
		"groups":        groups,
	}

	register(&dispatcher, collections)

	fs := http.FileServer(http.Dir("statics"))
	fsContent := http.FileServer(http.Dir("contents"))
	http.Handle("/statics/", http.StripPrefix("/statics/", fs))
	http.Handle("/contents/", http.StripPrefix("/contents/", fsContent))

	http.HandleFunc("/msg", func(w http.ResponseWriter, req *http.Request) {

		blocks := Blocks{}
		var err error
		w.Header().Add("Access-Control-Allow-Origin", "*")
		if req.Method == http.MethodGet {
			go func() {
				messages := forwardToMsgc(nil, []linebot.Message{})
				ids, err := getSubscriberIDs("msg", subscribers)
				if err != nil {
					log.Println(err)
					return
				}
				for _, id := range ids {
					if _, err := bot.PushMessage(id, messages[0]).Do(); err != nil {
						log.Println(err)
					}
				}
			}()
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

	http.HandleFunc("/users", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}

		if req.Header.Get("secret") != os.Getenv("CHANNEL_SECRET") {
			w.WriteHeader(403)
			w.Write([]byte("auth fail"))
			return
		}

		var user User
		err := json.NewDecoder(req.Body).Decode(&user)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("fail"))
			log.Println(err)
			return
		}
		_, err = users.UpsertId(user.ID, user)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("fail"))
			log.Println(err)
			return
		}
		w.Write([]byte("done"))
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
		ids, err := getSubscriberIDs("mg", subscribers)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("fail"))
			log.Println(err)
			return
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
		for _, _event := range events {
			event := _event
			b, _ := json.Marshal(event)
			log.Println("REQ: " + string(b))
			var group Group
			groupID := actionDispatcher.ExtractTargetID(&event)
			err := groups.FindId(groupID).One(&group)
			if err == mgo.ErrNotFound {
				group = Group{
					Users: map[string]interface{}{},
				}
			}
			if err != nil && err != mgo.ErrNotFound {
				log.Println(err.Error())
			} else {
				group.Users[event.Source.UserID] = true
				group.Type = event.Source.Type
				group.ID = groupID
				_, err := groups.UpsertId(groupID, group)
				if err != nil {
					log.Println(err.Error())
				}
			}
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
		messages = append(messages, linebot.NewImageMessage(bucketURLBase+id+".jpg", bucketURLBase+id+"_thumbnail.jpg"))
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
	url, err := upload(messageID+".jpg", "/tmp/", true)
	if err != nil {
		return "", "", err
	}
	thumbnailURL, err := upload(messageID+"_thumbnail.jpg", "/tmp/", true)
	return url, thumbnailURL, err
}

func decodeQR(message *linebot.ImageMessage, messages []linebot.Message) ([]linebot.Message, error) {
	path := "/tmp/" + message.ID + ".jpg"
	res, err := bot.GetMessageContent(message.ID).Do()
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(res.Content)

	// Create the file
	out, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	defer out.Close()

	// Writer the body to file
	thumbnail := resize.Resize(600, 0, img, resize.Lanczos3)

	err = jpeg.Encode(out, thumbnail, nil)
	if err != nil {
		return nil, err
	}

	output, err := exec.Command("node", "./js/qrcode.js", path).Output()
	if err != nil {
		return nil, err
	}

	if len(output) > 0 {
		messages = append(messages, linebot.NewTextMessage("QR: "+string(output)))
	}
	return messages, nil
}
