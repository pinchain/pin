package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pinchain/pinchain/testutil/keeper"
	"github.com/pinchain/pinchain/x/pin/keeper"
	"github.com/pinchain/pinchain/x/pin/types"
)

var (
	alice = sdk.AccAddress([]byte("alice_______________"))
	bob   = sdk.AccAddress([]byte("bob_________________"))
	carol = sdk.AccAddress([]byte("carol_______________"))

	alicePIN = "A7K-92XM"
	// The example PIN B4M-81QZ from the product brief contains "1", which the
	// PIN alphabet excludes, so Bob uses the nearest valid PIN.
	bobPIN = "B4M-89QZ"
)

func setup(t testing.TB) (keeper.Keeper, types.MsgServer, sdk.Context, *keepertest.MockBankKeeper) {
	k, ctx, bank := keepertest.PinKeeperWithBank(t)
	return k, keeper.NewMsgServerImpl(k), ctx, bank
}

func TestRegisterPIN(t *testing.T) {
	k, ms, ctx, _ := setup(t)
	ctx = ctx.WithBlockHeight(7)

	res, err := ms.RegisterPIN(ctx, types.NewMsgRegisterPIN(alice.String(), "a7k-92xm"))
	require.NoError(t, err)
	require.Equal(t, alicePIN, res.Pin)
	require.Equal(t, alice.String(), res.Owner)
	require.EqualValues(t, 7, res.CreationHeight)

	record, found := k.GetPINRecord(ctx, alicePIN)
	require.True(t, found)
	require.Equal(t, types.PIN_STATUS_ACTIVE, record.Status)
	require.Equal(t, alice.String(), record.Owner)
	require.EqualValues(t, 7, record.CreationHeight)

	require.Equal(t, []types.PINRecord{record}, k.GetPINsByOwner(ctx, alice))
	require.Empty(t, k.GetPINsByOwner(ctx, bob))
}

func TestRegisterPINRejectsDuplicates(t *testing.T) {
	_, ms, ctx, _ := setup(t)

	_, err := ms.RegisterPIN(ctx, types.NewMsgRegisterPIN(alice.String(), alicePIN))
	require.NoError(t, err)

	// same owner
	_, err = ms.RegisterPIN(ctx, types.NewMsgRegisterPIN(alice.String(), alicePIN))
	require.ErrorIs(t, err, types.ErrPINExists)

	// different owner, and lowercase input must not bypass the uniqueness check
	_, err = ms.RegisterPIN(ctx, types.NewMsgRegisterPIN(bob.String(), "a7k-92xm"))
	require.ErrorIs(t, err, types.ErrPINExists)
}

func TestRegisterPINRejectsMalformed(t *testing.T) {
	_, ms, ctx, _ := setup(t)

	for _, pin := range []string{"", "A7K92XM", "A7I-92XM", "A7K-92X!", "AAAA-AAAA"} {
		_, err := ms.RegisterPIN(ctx, &types.MsgRegisterPIN{Creator: alice.String(), Pin: pin})
		require.ErrorIs(t, err, types.ErrInvalidPIN, pin)
	}

	_, err := ms.RegisterPIN(ctx, &types.MsgRegisterPIN{Creator: "pin1invalid", Pin: alicePIN})
	require.ErrorIs(t, err, types.ErrInvalidPIN)
}

func TestRegisterPINRejectsExcludedDigits(t *testing.T) {
	_, ms, ctx, _ := setup(t)
	// "1" is excluded from the PIN alphabet together with 0, I, O and L.
	_, err := ms.RegisterPIN(ctx, types.NewMsgRegisterPIN(bob.String(), "B4M-81QZ"))
	require.ErrorIs(t, err, types.ErrInvalidPIN)
}

func TestResolvePIN(t *testing.T) {
	k, ms, ctx, _ := setup(t)
	_, err := ms.RegisterPIN(ctx, types.NewMsgRegisterPIN(bob.String(), bobPIN))
	require.NoError(t, err)

	record, owner, err := k.ResolvePIN(ctx, "b4m-89qz")
	require.NoError(t, err)
	require.Equal(t, bobPIN, record.Pin)
	require.Equal(t, bob, owner)

	_, _, err = k.ResolvePIN(ctx, "Z9Z-9999")
	require.ErrorIs(t, err, types.ErrPINNotFound)

	_, _, err = k.ResolvePIN(ctx, "nope")
	require.ErrorIs(t, err, types.ErrInvalidPIN)
}

func TestResolvePINRejectsRevoked(t *testing.T) {
	k, _, ctx, _ := setup(t)
	require.NoError(t, k.SetPINRecord(ctx, types.PINRecord{
		Pin:            bobPIN,
		Owner:          bob.String(),
		CreationHeight: 1,
		Status:         types.PIN_STATUS_REVOKED,
	}))

	_, _, err := k.ResolvePIN(ctx, bobPIN)
	require.ErrorIs(t, err, types.ErrPINNotActive)
}

func TestSendByPIN(t *testing.T) {
	_, ms, ctx, bank := setup(t)
	bank.Fund(alice, sdk.NewCoins(sdk.NewInt64Coin(types.BaseDenom, 100_000_000)))

	_, err := ms.RegisterPIN(ctx, types.NewMsgRegisterPIN(bob.String(), bobPIN))
	require.NoError(t, err)

	amount := sdk.NewCoins(sdk.NewInt64Coin(types.BaseDenom, 25_000_000))
	res, err := ms.SendByPIN(ctx, types.NewMsgSendByPIN(alice.String(), "b4m-89qz", amount))
	require.NoError(t, err)
	require.Equal(t, bob.String(), res.Recipient)

	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(types.BaseDenom, 75_000_000)), bank.SpendableCoins(ctx, alice))
	require.Equal(t, amount, bank.SpendableCoins(ctx, bob))
}

func TestSendByPINFailures(t *testing.T) {
	_, ms, ctx, bank := setup(t)
	bank.Fund(alice, sdk.NewCoins(sdk.NewInt64Coin(types.BaseDenom, 10)))
	_, err := ms.RegisterPIN(ctx, types.NewMsgRegisterPIN(bob.String(), bobPIN))
	require.NoError(t, err)

	one := sdk.NewCoins(sdk.NewInt64Coin(types.BaseDenom, 1))

	// unknown PIN
	_, err = ms.SendByPIN(ctx, types.NewMsgSendByPIN(alice.String(), "Z9Z-9999", one))
	require.ErrorIs(t, err, types.ErrPINNotFound)

	// malformed PIN
	_, err = ms.SendByPIN(ctx, &types.MsgSendByPIN{Sender: alice.String(), RecipientPin: "bad", Amount: one})
	require.ErrorIs(t, err, types.ErrInvalidPIN)

	// zero amount
	_, err = ms.SendByPIN(ctx, &types.MsgSendByPIN{
		Sender: alice.String(), RecipientPin: bobPIN,
		Amount: sdk.Coins{sdk.Coin{Denom: types.BaseDenom, Amount: sdkmath.NewInt(0)}},
	})
	require.ErrorIs(t, err, types.ErrInvalidAmount)

	// negative amount
	_, err = ms.SendByPIN(ctx, &types.MsgSendByPIN{
		Sender: alice.String(), RecipientPin: bobPIN,
		Amount: sdk.Coins{sdk.Coin{Denom: types.BaseDenom, Amount: sdkmath.NewInt(-5)}},
	})
	require.ErrorIs(t, err, types.ErrInvalidAmount)

	// insufficient balance
	_, err = ms.SendByPIN(ctx, types.NewMsgSendByPIN(alice.String(), bobPIN,
		sdk.NewCoins(sdk.NewInt64Coin(types.BaseDenom, 1_000_000))))
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient funds")

	// sender with no balance at all
	_, err = ms.SendByPIN(ctx, types.NewMsgSendByPIN(carol.String(), bobPIN, one))
	require.Error(t, err)

	// blocked recipient
	bank.Blocked[bob.String()] = true
	_, err = ms.SendByPIN(ctx, types.NewMsgSendByPIN(alice.String(), bobPIN, one))
	require.ErrorIs(t, err, types.ErrBlockedAddress)
}

func TestSendByPINOverflowBoundary(t *testing.T) {
	_, ms, ctx, bank := setup(t)
	bank.Fund(alice, sdk.NewCoins(sdk.NewInt64Coin(types.BaseDenom, 1)))
	_, err := ms.RegisterPIN(ctx, types.NewMsgRegisterPIN(bob.String(), bobPIN))
	require.NoError(t, err)

	// 2^256-ish amount: accepted as a well formed coin but unaffordable.
	huge, ok := sdkmath.NewIntFromString("115792089237316195423570985008687907853269984665640564039457584007913129639935")
	require.True(t, ok)
	_, err = ms.SendByPIN(ctx, types.NewMsgSendByPIN(alice.String(), bobPIN,
		sdk.Coins{sdk.NewCoin(types.BaseDenom, huge)}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient funds")
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(types.BaseDenom, 1)), bank.SpendableCoins(ctx, alice))
}

func TestQueryPIN(t *testing.T) {
	k, ms, ctx, _ := setup(t)
	_, err := ms.RegisterPIN(ctx, types.NewMsgRegisterPIN(bob.String(), bobPIN))
	require.NoError(t, err)

	res, err := k.PIN(ctx, &types.QueryPINRequest{Pin: "b4m-89qz"})
	require.NoError(t, err)
	require.Equal(t, bob.String(), res.Record.Owner)

	_, err = k.PIN(ctx, &types.QueryPINRequest{Pin: "Z9Z-9999"})
	require.Error(t, err)

	_, err = k.PIN(ctx, &types.QueryPINRequest{Pin: "bogus"})
	require.Error(t, err)

	byOwner, err := k.PINsByOwner(ctx, &types.QueryPINsByOwnerRequest{Owner: bob.String()})
	require.NoError(t, err)
	require.Len(t, byOwner.Records, 1)

	all, err := k.PINs(ctx, &types.QueryPINsRequest{})
	require.NoError(t, err)
	require.Len(t, all.Records, 1)
}
