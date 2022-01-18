package bot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

type Session struct {
	Command     string
	UserID      string
	CurrentStep int
	Data        interface{}
}

type Bot struct {
	client          *mixin.Client
	commands        map[string]Command
	supportedAssets map[string]string
	sessionMap      sync.Map
	pin             string
}

var commandMap = map[string]Command{}

func init() {
	for _, command := range commands {
		commandMap[command.Name()] = command
	}
}

func Init(keystore *mixin.Keystore, pin string) (*Bot, error) {
	supportedAssets, err := initAssets()
	if err != nil {
		return nil, err
	}

	client, err := mixin.NewFromKeystore(keystore)
	if err != nil {
		return nil, fmt.Errorf("bot: init error: %w", err)
	}

	return &Bot{
		client:          client,
		commands:        commandMap,
		supportedAssets: supportedAssets,
		pin:             pin,
	}, nil
}

func (b *Bot) Run(ctx context.Context) {
	for {
		if err := b.client.LoopBlaze(ctx, mixin.BlazeListenFunc(b.handleMessage)); err != nil {
			log.Printf("LoopBlaze: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *mixin.MessageView, userID string) error {
	if userID, _ := uuid.FromString(msg.UserID); userID == uuid.Nil {
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(msg.Data)
	if err != nil {
		return err
	}
	msg.Data = string(data)

	ctx = context.WithValue(ctx, botContextKey{}, b)
	if tmp, ok := b.sessionMap.Load(msg.UserID); ok {
		session := tmp.(*Session)
		command, ok := b.commands[session.Command]
		if ok {
			reply, err := command.Execute(ctx, session, msg)
			return b.handleCommandResult(ctx, msg, session, reply, err)
		} else {
			b.sessionMap.Delete(msg.UserID)
		}
	} else if msg.Category == mixin.MessageCategorySystemAccountSnapshot {
		reply, err := b.handleTransferMessage(ctx, msg, nil)
		return b.handleCommandResult(ctx, msg, nil, reply, err)
	}

	if msg.Category == mixin.MessageCategoryPlainText {
		msg.Data = strings.TrimSpace(msg.Data)
		ss := strings.SplitN(msg.Data, " ", 2)
		command, ok := b.commands[ss[0]]
		if ok {
			data := ""
			if len(ss) > 1 {
				data = ss[1]
			}
			msg.Data = data
			session := &Session{Command: command.Name(), UserID: msg.UserID}
			b.sessionMap.Store(msg.UserID, session)
			reply, err := command.Execute(ctx, session, msg)
			return b.handleCommandResult(ctx, msg, session, reply, err)
		}
	}

	payload := base64.StdEncoding.EncodeToString([]byte("Unsupported command, send '/help' to get all available commands."))
	id, _ := uuid.FromString(msg.MessageID)
	reply := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    msg.UserID,
		MessageID:      uuid.NewV5(id, "reply").String(),
		Category:       mixin.MessageCategoryPlainText,
		Data:           payload,
	}
	return b.client.SendMessage(ctx, reply)
}

func (b *Bot) handleTransferMessage(ctx context.Context, msg *mixin.MessageView, session *Session) (*mixin.MessageRequest, error) {
	if msg.UserID == b.client.ClientID {
		return nil, nil
	}

	var view mixin.TransferView
	if err := json.Unmarshal([]byte(msg.Data), &view); err != nil {
		return nil, err
	}

	if session == nil {
		return nil, b.transferBack(ctx, msg, &view)
	}

	incomingAsset, err := b.client.ReadAsset(ctx, view.AssetID)
	if err != nil {
		return nil, err
	}

	targetAsset := session.Data.(*mixin.Asset)
	if err := b.mtgSwap(ctx, msg.UserID, incomingAsset.AssetID, targetAsset.AssetID, view.Amount); err != nil {
		return nil, err
	}

	replyData := fmt.Sprintf(
		"%s -> %s, swap at 4swap.\nPlease check @7000103537 for swap result",
		incomingAsset.Symbol, targetAsset.Symbol,
	)

	id, _ := uuid.FromString(msg.MessageID)
	reply := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    msg.UserID,
		MessageID:      uuid.NewV5(id, "reply").String(),
		Category:       mixin.MessageCategoryPlainText,
		Data:           base64.StdEncoding.EncodeToString([]byte(replyData)),
	}

	return reply, nil
}

func (b *Bot) transferBack(ctx context.Context, msg *mixin.MessageView, view *mixin.TransferView) error {
	amount, err := decimal.NewFromString(view.Amount)
	if err != nil {
		return err
	}

	id, _ := uuid.FromString(msg.MessageID)
	input := &mixin.TransferInput{
		AssetID:    view.AssetID,
		OpponentID: msg.UserID,
		Amount:     amount,
		TraceID:    uuid.NewV5(id, "refund").String(),
		Memo:       "refund",
	}

	_, err = b.client.Transfer(ctx, input, b.pin)
	return err
}

func (b *Bot) handleCommandResult(ctx context.Context, msg *mixin.MessageView, session *Session, reply *mixin.MessageRequest, err error) error {
	if err != nil {
		if session != nil {
			b.sessionMap.Delete(session.UserID)
		}

		id, _ := uuid.FromString(msg.MessageID)
		b.client.SendMessage(ctx, &mixin.MessageRequest{
			ConversationID: msg.ConversationID,
			RecipientID:    msg.UserID,
			MessageID:      uuid.NewV5(id, "reply").String(),
			Category:       mixin.MessageCategoryPlainText,
			Data:           base64.StdEncoding.EncodeToString([]byte(err.Error())),
		})
		return err
	}
	if reply != nil {
		return b.client.SendMessage(ctx, reply)
	}
	if session != nil && session.Command == "" {
		b.sessionMap.Delete(session.UserID)
	}

	return nil
}
