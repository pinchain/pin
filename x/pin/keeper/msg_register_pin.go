package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pinchain/pinchain/x/pin/types"
)

// RegisterPIN registers a canonical PIN and binds it to the signer's address.
// The signer of the transaction is the only account that can become the owner:
// a PIN is an identifier, never a credential.
func (k msgServer) RegisterPIN(goCtx context.Context, msg *types.MsgRegisterPIN) (*types.MsgRegisterPINResponse, error) {
	if msg == nil {
		return nil, types.ErrInvalidPIN.Wrap("nil message")
	}

	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, types.ErrInvalidPIN.Wrapf("invalid creator address: %s", err)
	}

	pin, err := types.NormalizeAndValidatePIN(msg.Pin)
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.HasPIN(ctx, pin) {
		return nil, types.ErrPINExists.Wrap(pin)
	}

	record := types.PINRecord{
		Pin:            pin,
		Owner:          creator.String(),
		CreationHeight: ctx.BlockHeight(),
		Status:         types.PIN_STATUS_ACTIVE,
	}
	if err := k.SetPINRecord(ctx, record); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRegisterPIN,
		sdk.NewAttribute(types.AttributeKeyPIN, record.Pin),
		sdk.NewAttribute(types.AttributeKeyOwner, record.Owner),
		sdk.NewAttribute(types.AttributeKeyCreationHeight, strconv.FormatInt(record.CreationHeight, 10)),
	))

	return &types.MsgRegisterPINResponse{
		Pin:            record.Pin,
		Owner:          record.Owner,
		CreationHeight: record.CreationHeight,
	}, nil
}
