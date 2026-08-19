package types

const (
	// ModuleName defines the module name
	ModuleName = "pin"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_pin"
)

var (
	// ParamsKey is the store key of the module params.
	ParamsKey = []byte{0x00}

	// PINRecordKeyPrefix prefixes PIN -> PINRecord entries. This mapping is the
	// consensus source of truth for PIN resolution.
	PINRecordKeyPrefix = []byte{0x01}

	// OwnerPINKeyPrefix prefixes the (owner, pin) secondary index used for
	// reverse lookups. Values are empty.
	OwnerPINKeyPrefix = []byte{0x02}
)

// PINRecordKey returns the store key of a canonical PIN.
func PINRecordKey(pin string) []byte {
	return append(append([]byte{}, PINRecordKeyPrefix...), []byte(pin)...)
}

// OwnerPINKey returns the secondary index key for an owner/PIN pair.
func OwnerPINKey(owner []byte, pin string) []byte {
	key := append([]byte{}, OwnerPINKeyPrefix...)
	key = append(key, byte(len(owner)))
	key = append(key, owner...)
	return append(key, []byte(pin)...)
}

// OwnerPINPrefix returns the secondary index prefix for a single owner.
func OwnerPINPrefix(owner []byte) []byte {
	key := append([]byte{}, OwnerPINKeyPrefix...)
	key = append(key, byte(len(owner)))
	return append(key, owner...)
}
