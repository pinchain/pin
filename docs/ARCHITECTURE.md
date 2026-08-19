# Architecture

PINChain is a sovereign Cosmos SDK application chain: CometBFT for consensus
(Tendermint BFT proof-of-stake), Cosmos SDK v0.50 for the application, and one
custom module, `x/pin`.

```
CometBFT v0.38 (consensus, p2p, mempool)
        | ABCI
Cosmos SDK v0.50 baseapp
        |
+-------+---------------------------------------------+
| auth  bank  staking  distribution  slashing  gov    | standard modules
| mint  crisis  evidence  upgrade  consensus  genutil |
| authz  feegrant  group  nft  circuit  ibc/*         |
+-----------------------------------------------------+
| x/pin  (PIN identity: register, resolve, send-by-pin)|
+-----------------------------------------------------+
```

## Binary and layout

| Path | Contents |
| --- | --- |
| `cmd/pinchaind` | node/CLI entrypoint, built to `build/pinchaind` by `make build` |
| `app/` | app wiring (depinject app config, module ordering, account permissions) |
| `proto/pinchain/pin/` | `pin.proto`, `tx.proto`, `query.proto`, `genesis.proto` |
| `x/pin/types/` | generated types, PIN format rules, msgs, errors, store keys |
| `x/pin/keeper/` | state access, msg handlers, query server |
| `x/pin/module/` | AppModule, genesis import/export, CLI registration |
| `x/pin/client/cli/` | `tx pin ...` and `query pin ...` commands |
| `scripts/` | `singlenode.sh`, `localnet.sh`, `configure_node.py` |
| `tests/e2e/` | localnet E2E flow and security suite (build tag `e2e`) |

## Token and accounts

- Base denom `upin`, display denom `PIN`, `1 PIN = 1e6 upin`.
- Bech32 account prefix `pin` (`pin1...`), validator prefix `pinvaloper`.
- Staking bond denom, mint denom, gov deposit denom and crisis fee denom are all
  `upin` (set in the genesis produced by the localnet scripts).

## x/pin design decisions

- **The chain is the source of truth.** PIN records are consensus state in the
  `pin` store, replicated by every validator. PostgreSQL is not involved in any
  consensus path; it is reserved for a future read-only indexer.
- **No custom balances.** `MsgSendByPIN` resolves a PIN to an address and calls
  `x/bank`'s `SendCoins`. The module has no module account and cannot mint or burn.
  The bank keeper dependency is narrowed to `SpendableCoins`, `SendCoins` and
  `BlockedAddr` (`x/pin/types/expected_keepers.go`).
- **Authorization is standard SDK authorization.** Both messages declare their
  signer via `cosmos.msg.v1.signer`, so signature, account-number, sequence
  (replay protection) and fee checks all happen in the ante handler before a
  handler runs. `x/pin` adds no alternative auth path.
- **Normalization happens before every state touch**, in `ValidateBasic` and again
  in the keeper, so a lowercase PIN can neither create a duplicate record nor miss
  an existing one.
- **Resolution has a single implementation** (`Keeper.ResolvePIN`) shared by the
  query server and `MsgSendByPIN`, so CLI resolution and transfer routing can not
  diverge.

## Transaction flow (SendByPIN)

```
pinchaind tx pin send-by-pin B4M-89QZ 25PIN --from alice
  -> CLI builds MsgSendByPIN (amount parsed to upin), signs with alice's key
  -> broadcast to a validator's RPC
  -> ante handler: signature, sequence, fee deduction
  -> x/pin handler: validate amount, normalize + resolve PIN, blocked-addr check
  -> x/bank SendCoins(alice, bob, 25000000upin)
  -> events: send_by_pin + bank transfer
  -> block committed by 4/4 validators
```

## Networks

`scripts/localnet.sh` builds a four-validator network under `localnet/`, each
validator with its own home directory, node key, validator key, account and port
set, sharing one canonical genesis and connected through `persistent_peers`.
See `docs/LOCALNET.md`.

## Not built yet

Wallet, explorer, indexer/PostgreSQL, faucet, public testnet, monitoring and
upgrade tooling are out of scope for this milestone and are not present in the
repository.
