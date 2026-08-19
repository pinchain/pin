# Localnet

`make localnet` starts a reproducible four-validator PINChain network on one
machine (chain ID `pinchain-local-1`, denom `upin`). No Docker required.

## Commands

```bash
make build            # build build/pinchaind
make localnet         # (re)start the 4-validator network, waits for blocks
make localnet-status  # height / catching_up / voting power per validator
make localnet-stop    # stop all validators
make localnet-clean   # stop and delete all localnet state
make test-unit        # unit tests
make test-e2e         # E2E + security suite against a fresh localnet
./scripts/localnet.sh restart-node 2   # restart a single validator
./scripts/singlenode.sh               # single-node chain in the foreground
```

State lives in `localnet/validator-N/` and logs in `localnet/logs/`.
`localnet/` is git-ignored; validator keys are created in each node's `test`
keyring and never leave the machine.

## Topology

| Node | Home | RPC | P2P | gRPC | API |
| --- | --- | --- | --- | --- | --- |
| validator-1 | `localnet/validator-1` | 26657 | 26656 | 9090 | 1317 |
| validator-2 | `localnet/validator-2` | 26667 | 26666 | 9100 | 1327 |
| validator-3 | `localnet/validator-3` | 26677 | 26676 | 9110 | 1337 |
| validator-4 | `localnet/validator-4` | 26687 | 26686 | 9120 | 1347 |

Each validator has its own node identity, validator consensus key and account.
Every validator bonds `100000000000upin` (equal voting power 100000) from a
`1000000000000upin` genesis allocation. Validator 1 collects the four gentxs into
the canonical genesis, which is then copied to the other three homes;
`persistent_peers` is built from the real node IDs.

## Verifying the network

```bash
make localnet-status
```

Example output from a healthy network:

```
validator-1: height=3 catching_up=false voting_power=100000
validator-2: height=3 catching_up=false voting_power=100000
validator-3: height=3 catching_up=false voting_power=100000
validator-4: height=3 catching_up=false voting_power=100000
```

## Sending transactions

The genesis accounts are keys named `validator-N` in the matching node's `test`
keyring:

```bash
BIN=./build/pinchaind
HOME1=localnet/validator-1
FLAGS="--chain-id pinchain-local-1 --node tcp://127.0.0.1:26657 --keyring-backend test --home $HOME1 --fees 5000upin --gas 400000 -y"

$BIN keys add alice --keyring-backend test --home $HOME1
ALICE=$($BIN keys show alice -a --keyring-backend test --home $HOME1)

$BIN tx bank send validator-1 $ALICE 100000000upin --from validator-1 $FLAGS
$BIN tx pin register A7K-92XM --from alice $FLAGS
$BIN query pin resolve A7K-92XM --node tcp://127.0.0.1:26657 --output json
$BIN tx pin send-by-pin B4M-89QZ 25PIN --from alice $FLAGS
```

## Restart survival

```bash
./scripts/localnet.sh restart-node 2
make localnet-status   # validator-2 catches up, catching_up=false
```

The E2E suite automates this: it records the network height, restarts
validator-2, and requires it to reach height+3 with `catching_up=false`, then
queries PIN state through validator-2's RPC.

## E2E suite

```bash
make test-e2e          # cleans, starts a localnet, runs everything, stops it
KEEP_LOCALNET=1 make test-e2e   # leave the network running for inspection
```

`tests/e2e` drives the real `pinchaind` binary over RPC: account creation,
funding, PIN registration, resolution, a 25 PIN transfer by PIN with exact
balance assertions, transaction lookup in a committed block, validator restart,
and the adversarial cases in `tests/e2e/security_test.go`.

## Troubleshooting

- `make localnet` is idempotent: it stops running nodes and reuses existing state.
  Use `make localnet-clean` for a fresh chain (new genesis, new keys).
- Ports 26656-26687, 9090-9120 and 1317-1347 must be free.
- Logs: `tail -f localnet/logs/validator-2.log`.
