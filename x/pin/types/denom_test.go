package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pinchain/pinchain/x/pin/types"
)

func TestParseAmount(t *testing.T) {
	cases := []struct {
		input    string
		expected sdk.Coins
	}{
		{"25PIN", sdk.NewCoins(sdk.NewInt64Coin("upin", 25_000_000))},
		{"25pin", sdk.NewCoins(sdk.NewInt64Coin("upin", 25_000_000))},
		{"0.5PIN", sdk.NewCoins(sdk.NewInt64Coin("upin", 500_000))},
		{"25000000upin", sdk.NewCoins(sdk.NewInt64Coin("upin", 25_000_000))},
	}
	for _, tc := range cases {
		coins, err := types.ParseAmount(tc.input)
		require.NoError(t, err, tc.input)
		require.Equal(t, tc.expected, coins, tc.input)
	}
}

func TestParseAmountRejectsInvalid(t *testing.T) {
	for _, input := range []string{"", "PIN", "-25PIN", "25", "0.0000001PIN", "abcPIN", "25 PIN 25"} {
		_, err := types.ParseAmount(input)
		require.Error(t, err, input)
	}
}

func TestParseAmountZeroIsDroppedAndRejectedLater(t *testing.T) {
	// sdk.NewCoins drops zero coins; MsgSendByPIN then fails ValidateBasic.
	coins, err := types.ParseAmount("0PIN")
	require.NoError(t, err)
	require.True(t, coins.IsZero())

	msg := types.NewMsgSendByPIN(sdk.AccAddress("sender______________").String(), "A7K-92XM", coins)
	require.ErrorIs(t, msg.ValidateBasic(), types.ErrInvalidAmount)
}
