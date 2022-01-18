package bot

import (
	"context"
	"fmt"

	fswap "github.com/fox-one/4swap-sdk-go"
	mtg "github.com/fox-one/4swap-sdk-go/mtg"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

func (b *Bot) mtgSwap(ctx context.Context, receiverID, payAssetID, fillAssetID, amount string) error {
	// use the 4swap's MTG api endpoint
	fswap.UseEndpoint(fswap.MtgEndpoint)

	// read the mtg group
	group, err := fswap.ReadGroup(ctx)
	if err != nil {
		return err
	}

	// the ID to trace the orders at 4swap
	followID, _ := uuid.NewV4()

	// build a swap action, specified the swapping parameters
	action := mtg.SwapAction(
		// the user ID to receive the money
		receiverID,
		// an UUID get trace the order
		followID.String(),
		// the asset's ID you are swapping for.
		fillAssetID,
		// leave empty to let 4swap decide the routes.
		"",
		// the minimum amount of asset you will get.
		decimal.NewFromFloat(0.00000001),
	)

	// the action will be sent to 4swap in the memo
	memo, err := action.Encode(group.PublicKey)
	if err != nil {
		return err
	}

	// send a transaction to a multi-sign address which specified by `OpponentMultisig`
	// the OpponentMultisig.Receivers are the MTG group members of 4swap
	tx, err := b.client.Transaction(ctx, &mixin.TransferInput{
		AssetID: payAssetID,
		Amount:  decimal.RequireFromString(amount),
		TraceID: mixin.RandomTraceID(),
		Memo:    memo,
		OpponentMultisig: struct {
			Receivers []string `json:"receivers,omitempty"`
			Threshold uint8    `json:"threshold,omitempty"`
		}{
			Receivers: group.Members,
			Threshold: uint8(group.Threshold),
		},
	}, b.pin)

	if err != nil {
		return err
	}

	fmt.Println(tx)
	return nil
}
