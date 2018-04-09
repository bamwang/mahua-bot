package actionDispatcher

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/line/line-bot-sdk-go/linebot"
)

type ReplyActionHandler func(event *linebot.Event, context *Context) ([]linebot.Message, error)
type PushActionHandler func(event *linebot.Event, context *Context) ([]linebot.Message, string, error)

type Action interface {
	do(event *linebot.Event, client *linebot.Client, context *Context) (bool, *Context, error)
}

type ReplyAction struct {
	actionHandler ReplyActionHandler
}
type PushAction struct {
	IDs           []string
	actionHandler PushActionHandler
}
type ContextAction struct {
	rule           *rule
	activateAction *Action
}

type ActionDispatcher struct {
	client            *linebot.Client
	contextGroup      contextGroup
	defaultAction     Action
	defaultDoc        string
	keywordActionMap  map[string]Action
	keywordSourcesMap map[string][]linebot.EventSourceType
	keywordIDsMap     map[string][]string
	docMap            map[string]string
}

func New(client *linebot.Client) (actionDispatcher ActionDispatcher) {
	actionDispatcher.contextGroup = contextGroup{make(map[string]*Context), sync.RWMutex{}}
	actionDispatcher.keywordActionMap = make(map[string]Action)
	actionDispatcher.keywordSourcesMap = make(map[string][]linebot.EventSourceType)
	actionDispatcher.keywordIDsMap = make(map[string][]string)
	actionDispatcher.client = client
	return
}

func (d *ActionDispatcher) Dispatch(event *linebot.Event) {
	if event.Type != linebot.EventTypeMessage {
		return
	}

	var err error
	var replied bool
	// get context and do
	if context, has := d.contextGroup.Get(event.Source); has {
		if context.shouldInactvate(event.Message) {
			replied, _, err = context.rule.incativateAction.do(event, d.client, context)
			if err != nil {
				log.Println(err)
			}
			d.contextGroup.Delete(event.Source)
		} else {
			replied, _, err = context.rule.usualAction.do(event, d.client, context)
			if err != nil {
				log.Println(err)
			}
		}
	}
	if replied {
		return
	}

	// decide whether start a new conversion
	message, ok := event.Message.(*linebot.TextMessage)
	var words []string
	if ok {
		words = strings.Split(message.Text, " ")
	}
	if len(words) == 0 {
		d.doDefaultAction(event)
		return
	}

	keyword := words[0]
	action, hasAction := d.keywordActionMap[keyword]
	sourcesTypes := d.keywordSourcesMap[keyword]
	idsTypes := d.keywordIDsMap[keyword]
	typeMatched := len(sourcesTypes) == 0
	idMatched := len(idsTypes) == 0
	for _, eventType := range sourcesTypes {
		if event.Source.Type == eventType {
			typeMatched = true
			break
		}
	}
	for _, id := range idsTypes {
		if getTargetID(event) == id {
			idMatched = true
			break
		}
	}
	if hasAction && (typeMatched && idMatched) {
		var c *Context
		replied, c, err = action.do(event, d.client, nil)
		if err != nil {
			log.Println(err)
		}
		if c != nil {
			d.contextGroup.Put(event.Source, c)
		}
	}
	if replied {
		return
	}

	if words[0] == "m?" {
		d.replyDoc(event, d.client, nil)
	}

	// do default action
	d.doDefaultAction(event)
}

func (d *ActionDispatcher) doDefaultAction(event *linebot.Event) {
	if d.defaultAction != nil {
		action := d.defaultAction
		_, _, err := action.do(event, d.client, nil)
		if err != nil {
			log.Println(err)
		}
	}
}

func getTargetID(event *linebot.Event) (id string) {
	switch event.Source.Type {
	case linebot.EventSourceTypeGroup:
		id = event.Source.GroupID
	case linebot.EventSourceTypeRoom:
		id = event.Source.RoomID
	case linebot.EventSourceTypeUser:
		id = event.Source.UserID
	}
	return
}

func (d *ActionDispatcher) replyDoc(event *linebot.Event, client *linebot.Client, context *Context) {
	NewReplayAction(func(event *linebot.Event, context *Context) (messages []linebot.Message, err error) {
		text := "命令列表\n"
		if d.defaultDoc != "" {
			text += d.defaultDoc + "\n"
		}
		matchingID := getTargetID(event)
		for keyword := range d.keywordActionMap {
			skip := false
			if doc, has := d.docMap[keyword]; !has || doc == "" {
				skip = true
			}
			if ids, has := d.keywordIDsMap[keyword]; has && !skip {
				for _, id := range ids {
					if id != matchingID {
						skip = true
					}
				}
			}
			if sources, has := d.keywordSourcesMap[keyword]; has && !skip {
				for _, source := range sources {
					if event.Source.Type != source {
						skip = true
					}
				}
			}
			if skip {
				continue
			}
			text += fmt.Sprintf("%s: %s\n", keyword, d.docMap[keyword])
		}
		message := linebot.NewTextMessage(text)
		messages = append(messages, message)
		return
	}).do(event, client, context)
}

func NewReplayAction(actionHandler ReplyActionHandler) (action ReplyAction) {
	action.actionHandler = actionHandler
	return
}

func NewPushAction(actionHandler PushActionHandler) (action PushAction) {
	action.actionHandler = actionHandler
	return
}

func NewContextAction(inactivateCommands []string, activateAction, incativateAction, usualAction Action) (action ContextAction) {
	rule := rule{
		incativateAction:   incativateAction,
		usualAction:        usualAction,
		inactivateCommands: make(map[string]interface{}),
	}

	for _, word := range inactivateCommands {
		rule.inactivateCommands[word] = struct{}{}
	}
	action.rule = &rule
	action.activateAction = &activateAction
	return
}

// func (d *ActionDispatcher) RegisterContextActionWhithoutCommand(source *linebot.EventSource, action ContextAction) error {
// 	if action.rule == nil {
// 		err := errors.New("NO_INACTIVE_WORDS")
// 		return err
// 	}
// 	context := Context{
// 		rule: action.rule,
// 	}
// 	d.contextGroup.Put(source, context)
// 	return nil
// }

// RegisterWithType will register action to dispatcher.
// If sourceTypes is empty, dispatcher will ignore ids
func (d *ActionDispatcher) RegisterWithType(commands []string, sourceTypes []linebot.EventSourceType, doc string, action Action) {
	for _, command := range commands {
		d.keywordActionMap[command] = action
		d.keywordSourcesMap[command] = sourceTypes
		d.docMap[command] = doc
	}
}

// RegisterWithID will register action to dispatcher.
// If ids is empty, dispatcher will ignore ids
func (d *ActionDispatcher) RegisterWithID(commands []string, ids []string, doc string, action Action) {
	for _, command := range commands {
		d.keywordActionMap[command] = action
		d.keywordIDsMap[command] = ids
		d.docMap[command] = doc
	}
}

// RegisterDefaultAction will register default action
// action will responed
func (d *ActionDispatcher) RegisterDefaultAction(action Action, doc string) {
	d.defaultAction = action
	d.defaultDoc = doc
	fmt.Println(d.keywordActionMap)
	fmt.Println(d.keywordSourcesMap)
	fmt.Println(d.keywordIDsMap)
}

func (a PushAction) do(event *linebot.Event, client *linebot.Client, context *Context) (bool, *Context, error) {
	pushMessages, id, err := a.actionHandler(event, context)
	if err != nil {
		return false, context, err
	}
	if len(pushMessages) == 0 {
		return false, context, err
	}
	for _, m := range pushMessages {
		b, _ := m.MarshalJSON()
		log.Println(string(b))
	}
	_, err = client.PushMessage(id, pushMessages...).Do()
	return true, context, err
}

func (a ReplyAction) do(event *linebot.Event, client *linebot.Client, context *Context) (bool, *Context, error) {
	messages, err := a.actionHandler(event, context)
	if err != nil {
		return false, context, err
	}
	if len(messages) == 0 {
		return false, context, err
	}
	_, err = client.ReplyMessage(event.ReplyToken, messages...).Do()
	for _, m := range messages {
		b, _ := m.MarshalJSON()
		log.Println("RES: " + string(b))
	}
	return true, context, err
}

func (a ContextAction) do(event *linebot.Event, client *linebot.Client, _ *Context) (bool, *Context, error) {
	if a.rule == nil {
		err := errors.New("NO_INACTIVE_WORDS")
		return false, nil, err
	}
	context := Context{
		rule: a.rule,
	}
	// a.contextGroup.Put(event.Source, context)
	return (*a.activateAction).do(event, client, &context)
}
