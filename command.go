package bot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
)

var commands = []Command{
	echoCommandInstance,
	helpCommandInstance,
	priceCommandInstance,
	poemCommandInstance,
}

type Command interface {
	Name() string
	Desc() string
	Execute(context.Context, *Session, *mixin.MessageView) (*mixin.MessageRequest, error)
}

type botContextKey struct{}

type generalCommand struct {
	name string
	step int
	desc string
}

func (c *generalCommand) Name() string {
	return c.name
}

func (c *generalCommand) Desc() string {
	return c.desc
}

var echoCommandInstance = &echoCommand{
	generalCommand{
		name: "/echo",
		desc: "Echo message, it can be in the form of '/echo content', or send '/echo' first and then content.",
	},
}

type echoCommand struct {
	generalCommand
}

func (c *echoCommand) Execute(ctx context.Context, s *Session, msg *mixin.MessageView) (*mixin.MessageRequest, error) {
	data := msg.Data
	empty := s.CurrentStep == 0 && msg.Data == ""
	if empty {
		data = "Waiting to receive echo content..."
		s.CurrentStep += 1
	}

	id, _ := uuid.FromString(msg.MessageID)
	reply := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    msg.UserID,
		MessageID:      uuid.NewV5(id, "reply").String(),
		Category:       msg.Category,
		Data:           base64.StdEncoding.EncodeToString([]byte(data)),
	}

	if !empty {
		s.Command = ""
	}
	return reply, nil
}

var helpCommandInstance = &helpCommand{
	generalCommand{
		name: "/help",
		desc: "List all available commands.",
	},
}

type helpCommand struct {
	generalCommand
}

func (c *helpCommand) Execute(ctx context.Context, s *Session, msg *mixin.MessageView) (*mixin.MessageRequest, error) {
	id, _ := uuid.FromString(msg.MessageID)
	result := []string{}
	for _, command := range commands {
		result = append(result, fmt.Sprintf("%-10v%s", command.Name(), command.Desc()))
	}

	data := base64.StdEncoding.EncodeToString([]byte(strings.Join(result, "\n")))
	reply := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    msg.UserID,
		MessageID:      uuid.NewV5(id, "reply").String(),
		Category:       msg.Category,
		Data:           data,
	}

	s.Command = ""
	return reply, nil
}

var priceCommandInstance = &priceCommand{
	generalCommand{
		name: "/price",
		desc: "Query the prices of symbols.",
	},
}

type priceCommand struct {
	generalCommand
}

func (c *priceCommand) Execute(ctx context.Context, s *Session, msg *mixin.MessageView) (*mixin.MessageRequest, error) {
	data := ""
	empty := s.CurrentStep == 0 && msg.Data == ""
	if empty {
		data = "Waiting to receive symbols..."
		s.CurrentStep += 1
	} else {
		result := []string{}
		bot := ctx.Value(botContextKey{}).(*Bot)
		for _, str := range strings.Split(msg.Data, " ") {
			symbol := strings.ToUpper(strings.TrimSpace(str))
			assets, err := bot.getAssetBySymbol(ctx, symbol)
			if errors.Is(err, assetNotFoundError) {
				data = fmt.Sprintf("Symbol(%s) not found.", symbol)
				result = nil
				break
			} else if err != nil {
				return nil, err
			}
			result = append(result, fmt.Sprintf("%-10s $%s", symbol, assets.PriceUSD))
		}
		if len(result) != 0 {
			data = strings.Join(result, "\n")
		}
	}

	id, _ := uuid.FromString(msg.MessageID)
	reply := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    msg.UserID,
		MessageID:      uuid.NewV5(id, "reply").String(),
		Category:       msg.Category,
		Data:           base64.StdEncoding.EncodeToString([]byte(data)),
	}

	if !empty {
		s.Command = ""
	}
	return reply, nil
}

var poemCommandInstance = &poemCommand{
	generalCommand{
		name: "/poem",
		desc: "Return an ancient poem randomly.",
	},
}

type poemCommand struct {
	generalCommand
}

func (c *poemCommand) Execute(ctx context.Context, s *Session, msg *mixin.MessageView) (*mixin.MessageRequest, error) {
	resp, err := http.Get("https://v1.hitokoto.cn/?c=i")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID         int    `json:"id"`
		UUID       string `json:"uuid"`
		Hitokoto   string `json:"hitokoto"`
		Type       string `json:"type"`
		From       string `json:"from"`
		FromWho    string `json:"from_who"`
		Creator    string `json:"creator"`
		CreatorUID int    `json:"creator_uid"`
		Reviewer   int    `json:"reviewer"`
		CommitFrom string `json:"commit_from"`
		CreatedAt  string `json:"created_at"`
		Length     int    `json:"length"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	replyData := fmt.Sprintf("%s\n%s 《%s》", result.Hitokoto, result.FromWho, result.From)
	id, _ := uuid.FromString(msg.MessageID)
	reply := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    msg.UserID,
		MessageID:      uuid.NewV5(id, "reply").String(),
		Category:       msg.Category,
		Data:           base64.StdEncoding.EncodeToString([]byte(replyData)),
	}

	s.Command = ""
	return reply, err
}
