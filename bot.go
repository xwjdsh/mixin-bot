package bot

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
)

type Bot struct {
	client          *mixin.Client
	commands        map[string]Command
	supportedAssets map[string]string
	sessionMap      sync.Map
}

var commandMap = map[string]Command{}

func init() {
	for _, command := range commands {
		commandMap[command.Name()] = command
	}
}

func Init(keystore *mixin.Keystore) (*Bot, error) {
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

	if tmp, ok := b.sessionMap.Load(msg.UserID); ok {
		session := tmp.(*Session)
		command, ok := b.commands[session.Command]
		if ok {
			reply, err := command.Execute(session, msg)
			return b.handleCommandResult(ctx, session, reply, err)
		} else {
			b.sessionMap.Delete(msg.UserID)
		}
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
			reply, err := command.Execute(session, msg)
			return b.handleCommandResult(ctx, session, reply, err)
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

func (b *Bot) handleCommandResult(ctx context.Context, session *Session, reply *mixin.MessageRequest, err error) error {
	if err != nil {
		b.sessionMap.Delete(session.UserID)
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

func (b *Bot) getAssetBySymbol(ctx context.Context, symbol string) (*mixin.Asset, error) {
	symbol = strings.ToUpper(symbol)
	if assetID, found := b.supportedAssets[symbol]; found {
		return b.client.ReadAsset(ctx, assetID)
	}
	return nil, fmt.Errorf("Can't find asset (%s)", symbol)
}
