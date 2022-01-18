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
	"github.com/shopspring/decimal"
)

var commands = []Command{
	echoCommandInstance,
	helpCommandInstance,
	priceCommandInstance,
	poemCommandInstance,
	swapCommandInstance,
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
		result = append(result, fmt.Sprintf("%-10s%s", command.Name(), command.Desc()))
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
		step: 1,
		desc: "Query the prices of symbols.",
	},
}

type priceCommand struct {
	generalCommand
}

type priceData struct {
	symbols []*priceSymbol
}

type priceSymbol struct {
	symbol   string
	name     string
	priceUSD decimal.Decimal
}

func (c *priceCommand) Execute(ctx context.Context, s *Session, msg *mixin.MessageView) (*mixin.MessageRequest, error) {
	var replyData string
	pd := &priceData{}
	bot := ctx.Value(botContextKey{}).(*Bot)
	switch s.CurrentStep {
	case 0:
		for _, str := range strings.Split(msg.Data, " ") {
			symbol := strings.ToUpper(strings.TrimSpace(str))
			asset, err := bot.getAssetBySymbol(ctx, symbol)
			if errors.Is(err, assetNotFoundError) {
				replyData = fmt.Sprintf("Symbol(%s) not found.", symbol)
				pd.symbols = nil
				break
			} else if err != nil {
				return nil, err
			}
			pd.symbols = append(pd.symbols, &priceSymbol{symbol: symbol, name: asset.Name, priceUSD: asset.PriceUSD})
		}
		if len(pd.symbols) != 0 {
			replyData = "The base currence defaults to USD, enter 'Y' to ensure or enter the base coin you want (such as 'BTC'), enter 'N' to exit."
		}
		s.Data = pd
		s.CurrentStep += 1
	case 1:
		pd = s.Data.(*priceData)
		if input := strings.ToUpper(msg.Data); input == "Y" {
			tmp := []string{}
			for _, item := range pd.symbols {
				tmp = append(tmp, fmt.Sprintf("1 %s(%s) = %s USD", item.symbol, item.name, item.priceUSD))
			}
			replyData = strings.Join(tmp, "\n")
			s.CurrentStep += 1
			break
		} else if input == "N" {
			s.Command = ""
			return nil, nil
		}

		symbol := strings.ToUpper(msg.Data)
		assets, err := bot.getAssetBySymbol(ctx, symbol)
		if errors.Is(err, assetNotFoundError) {
			replyData = fmt.Sprintf("Symbol(%s) not found.", symbol)
			break
		}
		if err != nil {
			return nil, err
		}
		tmp := []string{}
		for _, item := range pd.symbols {
			tmp = append(tmp, fmt.Sprintf("1 %s(%s) ≈ %s %s(%s)", item.symbol, item.name, item.priceUSD.Div(assets.PriceUSD), assets.Symbol, assets.Name))
		}
		replyData = strings.Join(tmp, "\n")
		s.CurrentStep += 1
	}

	id, _ := uuid.FromString(msg.MessageID)
	reply := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    msg.UserID,
		MessageID:      uuid.NewV5(id, "reply").String(),
		Category:       msg.Category,
		Data:           base64.StdEncoding.EncodeToString([]byte(replyData)),
	}

	if s.CurrentStep > c.step {
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

var swapCommandInstance = &swapCommand{
	generalCommand{
		name: "/swap",
		desc: "Exchange your coin.",
	},
}

type swapCommand struct {
	generalCommand
}

func (c *swapCommand) Execute(ctx context.Context, s *Session, msg *mixin.MessageView) (*mixin.MessageRequest, error) {
	var replyData string
	category := mixin.MessageCategoryPlainText
	bot := ctx.Value(botContextKey{}).(*Bot)
	switch s.CurrentStep {
	case 0:
		symbol := strings.ToUpper(strings.TrimSpace(msg.Data))
		asset, err := bot.getAssetBySymbol(ctx, symbol)
		if errors.Is(err, assetNotFoundError) {
			replyData = fmt.Sprintf("Symbol(%s) not found.", symbol)
			s.Command = ""
		}
		if err != nil {
			return nil, err
		}
		s.Data = asset
		s.CurrentStep += 1
		replyData = fmt.Sprintf(`[{
    "label": "Swap to %s",
    "color": "#00BBFF",
    "action": "mixin://transfer/%s"
 	}]`, asset.Symbol, bot.client.ClientID)
		category = mixin.MessageCategoryAppButtonGroup
	case 1:
		s.Command = ""
		return bot.handleTransferMessage(ctx, msg, s)
	}

	id, _ := uuid.FromString(msg.MessageID)
	reply := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    msg.UserID,
		MessageID:      uuid.NewV5(id, "reply").String(),
		Category:       category,
		Data:           base64.StdEncoding.EncodeToString([]byte(replyData)),
	}
	return reply, nil
}
