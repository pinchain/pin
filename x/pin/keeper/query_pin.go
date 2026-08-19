package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pinchain/pinchain/x/pin/types"
)

// PIN resolves a PIN to its on-chain record.
func (k Keeper) PIN(goCtx context.Context, req *types.QueryPINRequest) (*types.QueryPINResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	pin, err := types.NormalizeAndValidatePIN(req.Pin)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	record, found := k.GetPINRecord(ctx, pin)
	if !found {
		return nil, status.Errorf(codes.NotFound, "pin %s is not registered", pin)
	}
	return &types.QueryPINResponse{Record: record}, nil
}

// PINsByOwner lists the PINs owned by an address.
func (k Keeper) PINsByOwner(goCtx context.Context, req *types.QueryPINsByOwnerRequest) (*types.QueryPINsByOwnerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	owner, err := sdk.AccAddressFromBech32(req.Owner)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryPINsByOwnerResponse{Records: k.GetPINsByOwner(ctx, owner)}, nil
}

// PINs lists every registered PIN record.
func (k Keeper) PINs(goCtx context.Context, req *types.QueryPINsRequest) (*types.QueryPINsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.PINRecordKeyPrefix)
	records := []types.PINRecord{}
	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		var record types.PINRecord
		if err := k.cdc.Unmarshal(value, &record); err != nil {
			return err
		}
		records = append(records, record)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryPINsResponse{Records: records, Pagination: pageRes}, nil
}
