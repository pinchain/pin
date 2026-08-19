package pin

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pinchain/pinchain/x/pin/keeper"
	"github.com/pinchain/pinchain/x/pin/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
	for _, record := range genState.PinRecords {
		if err := k.SetPINRecord(ctx, record); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	k.IteratePINRecords(ctx, func(record types.PINRecord) bool {
		genesis.PinRecords = append(genesis.PinRecords, record)
		return false
	})

	return genesis
}
