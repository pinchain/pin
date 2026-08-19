package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pinchain/pinchain/x/pin/types"
)

// MockBankKeeper is an in-memory stand-in for x/bank used by x/pin unit tests
// only. Production code always talks to the real bank keeper: see
// app/app_config.go wiring and tests/e2e for the on-chain assertions.
type MockBankKeeper struct {
	Balances map[string]sdk.Coins
	Blocked  map[string]bool
}

var _ types.BankKeeper = (*MockBankKeeper)(nil)

// NewMockBankKeeper returns an empty MockBankKeeper.
func NewMockBankKeeper() *MockBankKeeper {
	return &MockBankKeeper{
		Balances: map[string]sdk.Coins{},
		Blocked:  map[string]bool{},
	}
}

// Fund credits an account in the mock.
func (m *MockBankKeeper) Fund(addr sdk.AccAddress, coins sdk.Coins) {
	m.Balances[addr.String()] = m.Balances[addr.String()].Add(coins...)
}

// SpendableCoins implements types.BankKeeper.
func (m *MockBankKeeper) SpendableCoins(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	return m.Balances[addr.String()]
}

// SendCoins implements types.BankKeeper.
func (m *MockBankKeeper) SendCoins(_ context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	balance := m.Balances[from.String()]
	remaining, negative := balance.SafeSub(amt...)
	if negative {
		return fmt.Errorf("insufficient funds: %s is smaller than %s", balance, amt)
	}
	m.Balances[from.String()] = remaining
	m.Balances[to.String()] = m.Balances[to.String()].Add(amt...)
	return nil
}

// BlockedAddr implements types.BankKeeper.
func (m *MockBankKeeper) BlockedAddr(addr sdk.AccAddress) bool {
	return m.Blocked[addr.String()]
}
