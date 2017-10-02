package actionDispatcher

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/line/line-bot-sdk-go/linebot"
)

type ReplyActionHandler func(event linebot.Event, context *Context) ([]linebot.Message, error)
type PushActionHandler func(event linebot.Event, context *Context) ([]linebot.Message, string, error)

type Action interface {
	do(event linebot.Event, client *linebot.Client, context *Context) (bool, *Context, error)
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
	keywordActionMap  map[string]Action
	keywordSourcesMap map[string][]linebot.EventSourceType
	keywordIDsMap     map[string][]string
}

func New(client *linebot.Client) (actionDispatcher ActionDispatcher) {
	actionDispatcher.contextGroup = contextGroup{make(map[string]*Context), sync.RWMutex{}}
	actionDispatcher.keywordActionMap = make(map[string]Action)
	actionDispatcher.keywordSourcesMap = make(map[string][]linebot.EventSourceType)
	actionDispatcher.keywordIDsMap = make(map[string][]string)
	actionDispatcher.client = client
	return
}

func (d *ActionDispatcher) Dispatch(event linebot.Event) {
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

	message, ok := event.Message.(*linebot.TextMessage)
	if ok {
		action, has := d.keywordActionMap[message.Text]
		sourcesTypes := d.keywordSourcesMap[message.Text]
		idsTypes := d.keywordIDsMap[message.Text]
		typeMatched := len(sourcesTypes) == 0
		idMatched := len(idsTypes) == 0
		for _, eventType := range sourcesTypes {
			if event.Source.Type == eventType {
				typeMatched = true
				break
			}
		}
		for _, id := range idsTypes {
			if event.Source.UserID == id || event.Source.GroupID == id || event.Source.RoomID == id {
				idMatched = true
				break
			}
		}
		if has && (typeMatched && idMatched) {
			var c *Context
			if replied, c, err = action.do(event, d.client, nil); c != nil {
				if err != nil {
					log.Println(err)
				}
				d.contextGroup.Put(event.Source, c)
			}
		}
		if replied {
			return
		}
	}

	if d.defaultAction != nil {
		action := d.defaultAction
		_, _, err := action.do(event, d.client, nil)
		if err != nil {
			log.Println(err)
		}
	}
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
func (d *ActionDispatcher) RegisterWithType(commands []string, sourceTypes []linebot.EventSourceType, action Action) {
	for _, command := range commands {
		d.keywordActionMap[command] = action
		d.keywordSourcesMap[command] = sourceTypes
	}
}

// RegisterWithID will register action to dispatcher.
// If ids is empty, dispatcher will ignore ids
func (d *ActionDispatcher) RegisterWithID(commands []string, ids []string, action Action) {
	for _, command := range commands {
		d.keywordActionMap[command] = action
		d.keywordIDsMap[command] = ids
	}
}

// RegisterDefaultAction will register default action
// action will responed
func (d *ActionDispatcher) RegisterDefaultAction(action Action) {
	d.defaultAction = action
	fmt.Println(d.keywordActionMap)
	fmt.Println(d.keywordSourcesMap)
	fmt.Println(d.keywordIDsMap)
}

func (a PushAction) do(event linebot.Event, client *linebot.Client, context *Context) (bool, *Context, error) {
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

func (a ReplyAction) do(event linebot.Event, client *linebot.Client, context *Context) (bool, *Context, error) {
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

func (a ContextAction) do(event linebot.Event, client *linebot.Client, _ *Context) (bool, *Context, error) {
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
