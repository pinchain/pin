package pin_test

import (
	"testing"

	keepertest "github.com/pinchain/pinchain/testutil/keeper"
	"github.com/pinchain/pinchain/testutil/nullify"
	pin "github.com/pinchain/pinchain/x/pin/module"
	"github.com/pinchain/pinchain/x/pin/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.PinKeeper(t)
	pin.InitGenesis(ctx, k, genesisState)
	got := pin.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
