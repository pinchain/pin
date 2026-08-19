package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pinchain/pinchain/x/pin/types"
)

// SendByPIN resolves the recipient PIN to an address and performs a real
// x/bank transfer from the signer to that address. x/pin holds no balances of
// its own.
func (k msgServer) SendByPIN(goCtx context.Context, msg *types.MsgSendByPIN) (*types.MsgSendByPINResponse, error) {
	if msg == nil {
		return nil, types.ErrInvalidAmount.Wrap("nil message")
	}

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, types.ErrInvalidAmount.Wrapf("invalid sender address: %s", err)
	}

	if err := msg.Amount.Validate(); err != nil {
		return nil, types.ErrInvalidAmount.Wrapf("%s", err)
	}
	if !msg.Amount.IsAllPositive() {
		return nil, types.ErrInvalidAmount.Wrap("amount must be strictly positive")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	record, recipient, err := k.ResolvePIN(ctx, msg.RecipientPin)
	if err != nil {
		return nil, err
	}

	if k.bankKeeper.BlockedAddr(recipient) {
		return nil, types.ErrBlockedAddress.Wrap(record.Owner)
	}

	// x/bank enforces the balance check and updates state atomically; a failure
	// here aborts the whole message.
	if err := k.bankKeeper.SendCoins(ctx, sender, recipient, msg.Amount); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSendByPIN,
		sdk.NewAttribute(types.AttributeKeySender, sender.String()),
		sdk.NewAttribute(types.AttributeKeyRecipientPIN, record.Pin),
		sdk.NewAttribute(types.AttributeKeyRecipient, record.Owner),
		sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
	))

	return &types.MsgSendByPINResponse{Recipient: record.Owner}, nil
}
