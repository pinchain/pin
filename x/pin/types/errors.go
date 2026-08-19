package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/pin module sentinel errors
var (
	ErrInvalidSigner  = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidPIN     = sdkerrors.Register(ModuleName, 1101, "invalid pin")
	ErrPINExists      = sdkerrors.Register(ModuleName, 1102, "pin already registered")
	ErrPINNotFound    = sdkerrors.Register(ModuleName, 1103, "pin not found")
	ErrPINNotActive   = sdkerrors.Register(ModuleName, 1104, "pin is not active")
	ErrInvalidAmount  = sdkerrors.Register(ModuleName, 1105, "invalid amount")
	ErrBlockedAddress = sdkerrors.Register(ModuleName, 1106, "recipient address is not allowed to receive funds")
	ErrUnauthorized   = sdkerrors.Register(ModuleName, 1107, "unauthorized pin owner")
)
