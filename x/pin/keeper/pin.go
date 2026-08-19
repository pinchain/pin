package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pinchain/pinchain/x/pin/types"
)

// SetPINRecord writes a PIN record and its owner index entry.
func (k Keeper) SetPINRecord(ctx context.Context, record types.PINRecord) error {
	owner, err := sdk.AccAddressFromBech32(record.Owner)
	if err != nil {
		return err
	}
	bz, err := k.cdc.Marshal(&record)
	if err != nil {
		return err
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(types.PINRecordKey(record.Pin), bz)
	store.Set(types.OwnerPINKey(owner, record.Pin), []byte{})
	return nil
}

// GetPINRecord returns the record stored under a canonical PIN.
func (k Keeper) GetPINRecord(ctx context.Context, pin string) (types.PINRecord, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.PINRecordKey(pin))
	if bz == nil {
		return types.PINRecord{}, false
	}
	var record types.PINRecord
	k.cdc.MustUnmarshal(bz, &record)
	return record, true
}

// HasPIN reports whether a canonical PIN is already registered.
func (k Keeper) HasPIN(ctx context.Context, pin string) bool {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return store.Has(types.PINRecordKey(pin))
}

// ResolvePIN normalizes, validates and resolves a PIN to its owner address. It
// is the single resolution path used by both queries and MsgSendByPIN.
func (k Keeper) ResolvePIN(ctx context.Context, pin string) (types.PINRecord, sdk.AccAddress, error) {
	canonical, err := types.NormalizeAndValidatePIN(pin)
	if err != nil {
		return types.PINRecord{}, nil, err
	}
	record, found := k.GetPINRecord(ctx, canonical)
	if !found {
		return types.PINRecord{}, nil, types.ErrPINNotFound.Wrap(canonical)
	}
	if record.Status != types.PIN_STATUS_ACTIVE {
		return types.PINRecord{}, nil, types.ErrPINNotActive.Wrapf("%s has status %s", canonical, record.Status)
	}
	owner, err := sdk.AccAddressFromBech32(record.Owner)
	if err != nil {
		return types.PINRecord{}, nil, err
	}
	return record, owner, nil
}

// IteratePINRecords calls cb for every stored PIN record until cb returns true.
func (k Keeper) IteratePINRecords(ctx context.Context, cb func(types.PINRecord) (stop bool)) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.PINRecordKeyPrefix)
	iter := storetypes.KVStorePrefixIterator(store, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var record types.PINRecord
		k.cdc.MustUnmarshal(iter.Value(), &record)
		if cb(record) {
			return
		}
	}
}

// GetPINsByOwner returns every PIN record owned by addr, using the owner index.
func (k Keeper) GetPINsByOwner(ctx context.Context, addr sdk.AccAddress) []types.PINRecord {
	kvStore := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(kvStore, types.OwnerPINPrefix(addr))
	iter := storetypes.KVStorePrefixIterator(store, nil)
	defer iter.Close()

	records := []types.PINRecord{}
	for ; iter.Valid(); iter.Next() {
		if record, found := k.GetPINRecord(ctx, string(iter.Key())); found {
			records = append(records, record)
		}
	}
	return records
}
