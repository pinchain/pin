# PIN protocol

A PIN is a short, human-readable, on-chain alias for a PINChain account address.

## What a PIN is not

A PIN is **not** a private key, password, seed phrase, PIN code or signing
secret. It carries no spending authority: it is public data stored in consensus
state. Every state change is authorized by the account's private key through
standard Cosmos SDK signature verification. Knowing someone's PIN lets you send
tokens *to* them and nothing else.

## Format

```
XXX-XXXX
```

- 3 characters, a `-` separator, then 4 characters (8 characters total).
- Alphabet: `ABCDEFGHJKMNPQRSTUVWXYZ23456789` (31 symbols).
- Excluded on purpose because they are visually ambiguous: `0`, `1`, `I`, `O`, `L`.
- Input is normalized by trimming surrounding whitespace and upper-casing, so
  `a7k-92xm` and `A7K-92XM` are the same PIN. The canonical uppercase form is the
  only thing stored and the only key used for uniqueness.
- Keyspace: `31^7 = 27,512,614,111` PINs.

Note on the product brief: the example PIN `B4M-81QZ` contains `1`, which the
alphabet above excludes, so it is rejected. The tests and docs use `B4M-89QZ`
for Bob.

Implementation: `x/pin/types/format.go` (`NormalizePIN`, `ValidatePIN`,
`NormalizeAndValidatePIN`, `GeneratePIN`).

## PIN generation

`GeneratePIN` draws every character with `crypto/rand` via
`rand.Int(rand.Reader, len(alphabet))`. Timestamps, counters, block heights,
sequence numbers and any other predictable input are never used. Generation is
client-side only (`pinchaind query pin generate`); the chain only ever validates
and stores what a signed transaction submits.

## On-chain state

`PINRecord` (`proto/pinchain/pin/pin.proto`):

| Field | Type | Meaning |
| --- | --- | --- |
| `pin` | string | canonical PIN |
| `owner` | bech32 address | account that registered it |
| `creation_height` | int64 | block height of registration |
| `status` | enum | `PIN_STATUS_ACTIVE` / `PIN_STATUS_REVOKED` |

Keys (`x/pin/types/keys.go`):

- `0x01 | pin -> PINRecord` — primary mapping, source of truth for resolution.
- `0x02 | owner_bytes | pin -> pin` — index for "which PINs does this account own".

PIN records are part of the module genesis (`pin_records`) and are exported and
re-imported by `pinchaind export`.

## Messages

### MsgRegisterPIN

```
creator: bech32 address (signer)
pin:     string
```

Handler (`x/pin/keeper/msg_register_pin.go`):

1. The SDK verifies the signature, account number, sequence and fee before the
   handler runs.
2. Parse and validate `creator`.
3. Normalize and validate `pin`; malformed PINs are rejected.
4. Reject the PIN if it already exists — first registration wins, and there is no
   message that can reassign or steal an existing PIN.
5. Store the record with `owner = creator`, `creation_height = ctx.BlockHeight()`,
   `status = ACTIVE`, plus the owner index entry.
6. Emit a `register_pin` event.

### MsgSendByPIN

```
sender:        bech32 address (signer)
recipient_pin: string
amount:        repeated Coin
```

Handler (`x/pin/keeper/msg_send_by_pin.go`):

```
signature verified by the SDK ante handler (fee charged)
  -> parse sender
  -> validate amount (well formed, all positive)
  -> normalize recipient PIN
  -> resolve PIN to a record (must exist and be ACTIVE)
  -> reject blocked recipient addresses
  -> x/bank SendCoins(sender, owner, amount)
  -> emit send_by_pin event
  -> commit
```

Balances live entirely in `x/bank`. The `x/pin` module keeps no balance state and
never mints, burns or holds funds; it has no module account.

## Queries

| Query | gRPC / REST | CLI |
| --- | --- | --- |
| resolve one PIN | `/pinchain/pin/v1/pins/{pin}` | `pinchaind query pin resolve A7K-92XM` |
| PINs of an owner | `/pinchain/pin/v1/owners/{owner}/pins` | `pinchaind query pin by-owner pin1...` |
| all PINs (paginated) | `/pinchain/pin/v1/pins` | `pinchaind query pin list` |
| module params | `/pinchain/pin/v1/params` | `pinchaind query pin params` |

All CLI commands accept `--output json`.

## CLI examples

```bash
pinchaind tx pin register A7K-92XM --from alice --chain-id pinchain-local-1 \
  --keyring-backend test --fees 5000upin --gas 400000 -y

pinchaind query pin resolve A7K-92XM --output json

pinchaind tx pin send-by-pin B4M-89QZ 25PIN --from alice \
  --chain-id pinchain-local-1 --keyring-backend test --fees 5000upin --gas 400000 -y
```

Amounts accept display units (`25PIN`, `0.5PIN`) and base units (`25000000upin`).
Display amounts finer than `1upin` are rejected rather than silently truncated.

## Not implemented yet

- PIN revocation / transfer messages (the `REVOKED` status exists in state and is
  honoured by resolution, but no message sets it).
- PIN expiry, renewal, pricing or reservation policy.
- Fees specific to registration beyond the standard transaction fee.
