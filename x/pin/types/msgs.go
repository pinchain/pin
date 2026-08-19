package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg              = (*MsgRegisterPIN)(nil)
	_ sdk.HasValidateBasic = (*MsgRegisterPIN)(nil)
	_ sdk.Msg              = (*MsgSendByPIN)(nil)
	_ sdk.HasValidateBasic = (*MsgSendByPIN)(nil)
)

// NewMsgRegisterPIN creates a MsgRegisterPIN with a normalized PIN.
func NewMsgRegisterPIN(creator, pin string) *MsgRegisterPIN {
	return &MsgRegisterPIN{Creator: creator, Pin: NormalizePIN(pin)}
}

// ValidateBasic performs stateless validation of MsgRegisterPIN.
func (msg *MsgRegisterPIN) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.Wrapf(ErrInvalidPIN, "invalid creator address (%s)", err)
	}
	// The PIN is normalized here as well so that a message built outside of the
	// CLI (lowercase input, padded input) is judged on its canonical form.
	if _, err := NormalizeAndValidatePIN(msg.Pin); err != nil {
		return err
	}
	return nil
}

// NewMsgSendByPIN creates a MsgSendByPIN with a normalized recipient PIN.
func NewMsgSendByPIN(sender, recipientPIN string, amount sdk.Coins) *MsgSendByPIN {
	return &MsgSendByPIN{
		Sender:       sender,
		RecipientPin: NormalizePIN(recipientPIN),
		Amount:       amount,
	}
}

// ValidateBasic performs stateless validation of MsgSendByPIN.
func (msg *MsgSendByPIN) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAmount, "invalid sender address (%s)", err)
	}
	if _, err := NormalizeAndValidatePIN(msg.RecipientPin); err != nil {
		return err
	}
	// Coins.Validate rejects zero/negative amounts, unsorted denoms and duplicates.
	if err := msg.Amount.Validate(); err != nil {
		return sdkerrors.Wrapf(ErrInvalidAmount, "%s", err)
	}
	if !msg.Amount.IsAllPositive() {
		return ErrInvalidAmount.Wrap("amount must be strictly positive")
	}
	return nil
}
