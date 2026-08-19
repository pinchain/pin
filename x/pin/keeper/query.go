package keeper

import (
	"github.com/pinchain/pinchain/x/pin/types"
)

var _ types.QueryServer = Keeper{}
