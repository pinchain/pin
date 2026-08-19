package types_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pinchain/pinchain/x/pin/types"
)

var (
	testAddr  = sdk.AccAddress([]byte("alice_______________")).String()
	testCoins = sdk.NewCoins(sdk.NewInt64Coin(types.BaseDenom, 25_000_000))
)

func TestMsgRegisterPINValidateBasic(t *testing.T) {
	require.NoError(t, types.NewMsgRegisterPIN(testAddr, "a7k-92xm").ValidateBasic())

	require.ErrorIs(t, types.NewMsgRegisterPIN("not-an-address", "A7K-92XM").ValidateBasic(), types.ErrInvalidPIN)
	require.ErrorIs(t, types.NewMsgRegisterPIN(testAddr, "A7I-92XM").ValidateBasic(), types.ErrInvalidPIN)
	require.ErrorIs(t, types.NewMsgRegisterPIN(testAddr, "").ValidateBasic(), types.ErrInvalidPIN)
}

func TestMsgRegisterPINNormalizesPIN(t *testing.T) {
	require.Equal(t, "A7K-92XM", types.NewMsgRegisterPIN(testAddr, " a7k-92xm ").Pin)
}

func TestMsgSendByPINValidateBasic(t *testing.T) {
	require.NoError(t, types.NewMsgSendByPIN(testAddr, "b4m-89qz", testCoins).ValidateBasic())

	require.ErrorIs(t,
		types.NewMsgSendByPIN("not-an-address", "B4M-89QZ", testCoins).ValidateBasic(),
		types.ErrInvalidAmount)
	require.ErrorIs(t,
		types.NewMsgSendByPIN(testAddr, "B4M-8", testCoins).ValidateBasic(),
		types.ErrInvalidPIN)
	require.ErrorIs(t,
		types.NewMsgSendByPIN(testAddr, "B4M-89QZ", sdk.Coins{}).ValidateBasic(),
		types.ErrInvalidAmount)
	require.ErrorIs(t,
		types.NewMsgSendByPIN(testAddr, "B4M-89QZ", sdk.Coins{sdk.Coin{Denom: types.BaseDenom, Amount: sdkmath.NewInt(0)}}).ValidateBasic(),
		types.ErrInvalidAmount)
	require.ErrorIs(t,
		types.NewMsgSendByPIN(testAddr, "B4M-89QZ", sdk.Coins{sdk.Coin{Denom: types.BaseDenom, Amount: sdkmath.NewInt(-1)}}).ValidateBasic(),
		types.ErrInvalidAmount)
}

func TestMsgSendByPINRejectsMalformedDenom(t *testing.T) {
	msg := types.NewMsgSendByPIN(testAddr, "B4M-89QZ",
		sdk.Coins{sdk.Coin{Denom: "!!!", Amount: sdkmath.NewInt(1)}})
	require.ErrorIs(t, msg.ValidateBasic(), types.ErrInvalidAmount)
}
