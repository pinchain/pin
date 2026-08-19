package types

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// BaseDenom is the smallest unit of the native token and the only denom
	// stored in state.
	BaseDenom = "upin"
	// DisplayDenom is the human readable unit of the native token.
	DisplayDenom = "PIN"
	// DisplayExponent is the number of base units in one display unit:
	// 1 PIN = 10^6 upin.
	DisplayExponent = 6
)

// ParseAmount parses a CLI amount, accepting both base units ("25000000upin")
// and display units ("25PIN", "0.5pin"), and returns coins denominated in base
// units.
func ParseAmount(amount string) (sdk.Coins, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return nil, fmt.Errorf("empty amount")
	}

	coins := sdk.NewCoins()
	for _, part := range strings.Split(amount, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty amount in %q", amount)
		}
		coin, err := parseSingleAmount(part)
		if err != nil {
			return nil, err
		}
		coins = coins.Add(coin)
	}
	return coins, nil
}

func parseSingleAmount(part string) (sdk.Coin, error) {
	idx := strings.IndexFunc(part, func(r rune) bool {
		return !(r >= '0' && r <= '9') && r != '.'
	})
	if idx <= 0 {
		return sdk.Coin{}, fmt.Errorf("invalid amount %q", part)
	}
	value, denom := part[:idx], part[idx:]

	if !strings.EqualFold(denom, DisplayDenom) {
		coin, err := sdk.ParseCoinNormalized(part)
		if err != nil {
			return sdk.Coin{}, err
		}
		return coin, nil
	}

	dec, err := math.LegacyNewDecFromStr(value)
	if err != nil {
		return sdk.Coin{}, fmt.Errorf("invalid amount %q: %w", part, err)
	}
	base := dec.Mul(math.LegacyNewDec(10).Power(DisplayExponent))
	if !base.TruncateDec().Equal(base) {
		return sdk.Coin{}, fmt.Errorf("amount %q is more precise than 1%s", part, BaseDenom)
	}
	return sdk.NewCoin(BaseDenom, base.TruncateInt()), nil
}
