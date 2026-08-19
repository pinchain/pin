# PINChain

PINChain is a sovereign Cosmos SDK / CometBFT Layer-1 with a native human-readable
identity layer: every account can register a short PIN such as `A7K-92XM`, and
tokens can be sent to that PIN instead of a bech32 address.

```
pinchaind tx pin register A7K-92XM --from alice
pinchaind query pin resolve A7K-92XM
pinchaind tx pin send-by-pin B4M-89QZ 25PIN --from alice
```

A PIN is a public alias, **not** a private key, password or seed phrase.
Authorization always comes from the account's key through standard Cosmos SDK
signature verification.

## What works today

- `pinchaind` binary (`make build`), Cosmos SDK v0.50.14 + CometBFT v0.38.17.
- Proof-of-stake consensus with a reproducible four-validator localnet
  (`make localnet`) producing real blocks, and surviving a validator restart.
- Native `x/pin` module: on-chain `PIN -> address` records with owner, creation
  height and status; `MsgRegisterPIN`; `MsgSendByPIN` backed by the real `x/bank`
  module; resolve / by-owner / list / params queries over gRPC, REST and CLI.
- Unit tests for PIN format, generation, keeper state and message handlers, plus
  an E2E suite and an adversarial security suite that run against a live
  four-validator network (`make test-e2e`).

Not built yet: wallet, explorer, indexer/PostgreSQL, faucet, public testnet,
mainnet, PIN revocation/transfer.

## Token

| | |
| --- | --- |
| display denom | `PIN` |
| base denom | `upin` |
| conversion | `1 PIN = 1 000 000 upin` |
| address prefix | `pin1...` |
| local chain ID | `pinchain-local-1` |

## Quick start

```bash
make build                 # -> build/pinchaind
make localnet              # 4 validators on one machine, waits for blocks
make localnet-status       # height / catching_up / voting power per validator
make test-unit             # unit tests
make test-e2e              # E2E + security suite on a fresh localnet
make localnet-clean        # tear everything down
```

Then, using validator-1's keyring:

```bash
BIN=./build/pinchaind
FLAGS="--home localnet/validator-1 --chain-id pinchain-local-1 \
  --node tcp://127.0.0.1:26657 --keyring-backend test --fees 5000upin --gas 400000 -y"

$BIN keys add alice --keyring-backend test --home localnet/validator-1
ALICE=$($BIN keys show alice -a --keyring-backend test --home localnet/validator-1)
$BIN tx bank send validator-1 $ALICE 100000000upin --from validator-1 $FLAGS

$BIN tx pin register A7K-92XM --from alice $FLAGS
$BIN query pin resolve A7K-92XM --node tcp://127.0.0.1:26657 --output json
$BIN tx pin send-by-pin A7K-92XM 25PIN --from validator-1 $FLAGS
```

A single node is also available: `./scripts/singlenode.sh`.

## PIN format

`XXX-XXXX` over the alphabet `ABCDEFGHJKMNPQRSTUVWXYZ23456789` — uppercase A-Z and
2-9, excluding the ambiguous `0`, `1`, `I`, `O`, `L`. Input is normalized
(trimmed, upper-cased) before validation and storage, so `a7k-92xm` and
`A7K-92XM` are the same PIN. Candidate PINs are generated with `crypto/rand`
(`pinchaind query pin generate`).

## Documentation

| Document | Contents |
| --- | --- |
| [docs/VERSIONS.md](docs/VERSIONS.md) | pinned dependency versions and why |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | app layout, modules, design decisions |
| [docs/PIN_PROTOCOL.md](docs/PIN_PROTOCOL.md) | PIN format, state, messages, queries |
| [docs/LOCALNET.md](docs/LOCALNET.md) | localnet topology, commands, troubleshooting |
| [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) | enforced properties, known risks |

## Development

```bash
make govet          # vet (excluding generated api/)
make lint           # golangci-lint
make proto-gen      # regenerate protobuf code (requires ignite)
```

Localnet state under `localnet/` is git-ignored and its keys use the unencrypted
`test` keyring backend: development only.
