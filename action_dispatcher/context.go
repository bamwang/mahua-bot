package actionDispatcher

import (
	"crypto/sha1"
	"encoding/base64"
	"sync"

	"github.com/line/line-bot-sdk-go/linebot"
)

type Context struct {
	rule *rule
	data interface{}
}

type rule struct {
	inactivateCommands            map[string]interface{}
	incativateAction, usualAction Action
}

type contextGroup struct {
	contextMap map[string]*Context
	// count       int
	// limit      int
	lock sync.RWMutex
}

func (c *contextGroup) ForceDestory(source *linebot.EventSource) {
	key := convertIDstoKey(source)
	c.delete(key)
}

func (c *contextGroup) Put(source *linebot.EventSource, context *Context) {
	c.put(convertIDstoKey(source), context)
}

func (c *contextGroup) put(key string, context *Context) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.contextMap[key] = context
}

func (c *contextGroup) Get(source *linebot.EventSource) (*Context, bool) {
	return c.get(convertIDstoKey(source))
}

func (c *contextGroup) get(key string) (*Context, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	context, has := c.contextMap[key]
	return context, has
}

func (c *contextGroup) Delete(source *linebot.EventSource) {
	c.delete(convertIDstoKey(source))
}

func (c *contextGroup) delete(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.contextMap, key)
}

func (c *Context) SetData(data interface{}) {
	c.data = data
}

func (c *Context) GetData() interface{} {
	return c.data
}

func (c *Context) shouldInactvate(message linebot.Message) bool {
	mess, ok := message.(*linebot.TextMessage)
	if !ok {
		return false
	}
	_, has := c.rule.inactivateCommands[mess.Text]
	return has
}

func convertIDstoKey(source *linebot.EventSource) string {
	hasher := sha1.New()
	var id string
	switch source.Type {
	case linebot.EventSourceTypeGroup:
		id = source.GroupID
	case linebot.EventSourceTypeRoom:
		id = source.RoomID
	case linebot.EventSourceTypeUser:
		id = source.UserID
	}
	hasher.Write([]byte(id))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}
