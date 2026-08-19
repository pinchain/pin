# Versions

Pinned, mutually compatible versions used by PINChain. Everything below is what
the repository actually builds and tests against today.

| Component | Version | Notes |
| --- | --- | --- |
| Go | 1.23.6 | `go.mod` declares `go 1.23` |
| Cosmos SDK | v0.50.14 | latest stable v0.50 line |
| CometBFT | v0.38.17 | consensus engine matching SDK v0.50 |
| cosmossdk.io/store | v1.1.1 | |
| cosmossdk.io/math | v1.5.0 | |
| cosmossdk.io/api | v0.7.6 | |
| cosmossdk.io/log | v1.5.0 | |
| ibc-go | v8.5.2 | scaffolded, not exercised by the milestone tests |
| Ignite CLI | v28.11.2 | used once for scaffolding and proto generation |
| Docker | 27.4.1 | available in the dev environment; not required by `make localnet` |
| PostgreSQL | not used yet | reserved for the future indexer (derived data only) |

## Version selection notes

- Cosmos SDK v0.50.x is the current stable line and requires CometBFT v0.38.x;
  mixing v0.38 with SDK v0.47 or v0.53 is not supported.
- OpenTelemetry is pinned to v1.32.0. Later releases (v1.45.0) require Go >= 1.25,
  which is newer than the toolchain used here.
- `ibc-go` v8.5.2 is the release compatible with SDK v0.50; IBC is scaffolded but
  outside the Phase 1-3 milestone scope.

## Chain identifiers

| Purpose | Chain ID | Status |
| --- | --- | --- |
| local development | `pinchain-local-1` | implemented (`make localnet`) |
| public testnet | `pinchain-testnet-1` | reserved, not launched |
| mainnet | `pinchain-1` | reserved, not launched |

## Token

- Display denom: `PIN`
- Base denom: `upin`
- `1 PIN = 1_000_000 upin`
