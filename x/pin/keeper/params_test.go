package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pinchain/pinchain/testutil/keeper"
	"github.com/pinchain/pinchain/x/pin/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.PinKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
