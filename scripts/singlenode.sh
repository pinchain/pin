#!/usr/bin/env bash
# Single-node PINChain devnet: used as the Phase 1 smoke test (real blocks from one validator).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${BIN:-$ROOT_DIR/build/pinchaind}"
CHAIN_ID="${CHAIN_ID:-pinchain-local-1}"
HOME_DIR="${HOME_DIR:-$ROOT_DIR/localnet/single}"
DENOM="upin"
KEYRING="test"

rm -rf "$HOME_DIR"
mkdir -p "$HOME_DIR"

$BIN init single --chain-id "$CHAIN_ID" --home "$HOME_DIR" --default-denom "$DENOM" >/dev/null 2>&1

GENESIS="$HOME_DIR/config/genesis.json"
tmp=$(mktemp)
jq --arg d "$DENOM" '
  .app_state.staking.params.bond_denom = $d
  | .app_state.crisis.constant_fee.denom = $d
  | .app_state.gov.params.min_deposit[0].denom = $d
  | .app_state.mint.params.mint_denom = $d
' "$GENESIS" > "$tmp" && mv "$tmp" "$GENESIS"

$BIN keys add validator --keyring-backend "$KEYRING" --home "$HOME_DIR" >/dev/null 2>&1
ADDR=$($BIN keys show validator -a --keyring-backend "$KEYRING" --home "$HOME_DIR")
$BIN genesis add-genesis-account "$ADDR" "1000000000000$DENOM" --keyring-backend "$KEYRING" --home "$HOME_DIR"
$BIN genesis gentx validator "100000000000$DENOM" --chain-id "$CHAIN_ID" --keyring-backend "$KEYRING" --home "$HOME_DIR" >/dev/null 2>&1
$BIN genesis collect-gentxs --home "$HOME_DIR" >/dev/null 2>&1
$BIN genesis validate-genesis --home "$HOME_DIR"

echo "single node initialized at $HOME_DIR (validator: $ADDR)"
exec $BIN start --home "$HOME_DIR" --minimum-gas-prices "0.001$DENOM"
