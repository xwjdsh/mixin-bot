package bot

import (
	"context"
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
	supportedAssets map[string]string
	sessionMap      sync.Map
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

	id, _ := uuid.FromString(msg.MessageID)

	reply := &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    msg.UserID,
		MessageID:      uuid.NewV5(id, "reply").String(),
		Category:       msg.Category,
		Data:           msg.Data,
	}

	return b.client.SendMessage(ctx, reply)
}

func (b *Bot) getAssetBySymbol(ctx context.Context, symbol string) (*mixin.Asset, error) {
	symbol = strings.ToUpper(symbol)
	if assetID, found := b.supportedAssets[symbol]; found {
		return b.client.ReadAsset(ctx, assetID)
	}
	return nil, fmt.Errorf("Can't find asset (%s)", symbol)
}
