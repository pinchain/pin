package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		PinRecords: []PINRecord{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	seen := make(map[string]struct{}, len(gs.PinRecords))
	for _, record := range gs.PinRecords {
		if err := ValidatePIN(record.Pin); err != nil {
			return err
		}
		if _, duplicate := seen[record.Pin]; duplicate {
			return ErrPINExists.Wrap(record.Pin)
		}
		seen[record.Pin] = struct{}{}

		if _, err := sdk.AccAddressFromBech32(record.Owner); err != nil {
			return err
		}
		if record.Status == PIN_STATUS_UNSPECIFIED {
			return ErrPINNotActive.Wrapf("pin %s has unspecified status", record.Pin)
		}
		if record.CreationHeight < 0 {
			return ErrInvalidPIN.Wrapf("pin %s has negative creation height", record.Pin)
		}
	}

	return gs.Params.Validate()
}
