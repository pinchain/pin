#!/usr/bin/env bash
# Reproducible 4-validator PINChain localnet driven by separate pinchaind processes.
#
#   ./scripts/localnet.sh start     init (if needed) and start validator-1..4
#   ./scripts/localnet.sh stop      stop all validators
#   ./scripts/localnet.sh status    print height/voting info for each validator
#   ./scripts/localnet.sh clean     stop and delete all localnet state
#   ./scripts/localnet.sh restart-node <n>   stop and start a single validator
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${BIN:-$ROOT_DIR/build/pinchaind}"
CHAIN_ID="${CHAIN_ID:-pinchain-local-1}"
NET_DIR="${NET_DIR:-$ROOT_DIR/localnet}"
DENOM="${DENOM:-upin}"
KEYRING="test"
NUM_VALIDATORS=4

# Genesis allocations (in upin; 1 PIN = 1_000_000 upin).
VAL_BALANCE="1000000000000$DENOM" # 1,000,000 PIN
VAL_STAKE="100000000000$DENOM"    #   100,000 PIN bonded

node_home()  { echo "$NET_DIR/validator-$1"; }
p2p_port()   { echo $((26656 + ($1 - 1) * 10)); }
rpc_port()   { echo $((26657 + ($1 - 1) * 10)); }
grpc_port()  { echo $((9090 + ($1 - 1) * 10)); }
api_port()   { echo $((1317 + ($1 - 1) * 10)); }
pprof_port() { echo $((6060 + ($1 - 1) * 10)); }

require_bin() {
  if [[ ! -x "$BIN" ]]; then
    echo "pinchaind not found at $BIN — run 'make build' first" >&2
    exit 1
  fi
}

init_network() {
  echo "==> initializing $NUM_VALIDATORS-validator localnet ($CHAIN_ID) in $NET_DIR"
  rm -rf "$NET_DIR"
  mkdir -p "$NET_DIR"

  for i in $(seq 1 $NUM_VALIDATORS); do
    local home; home=$(node_home "$i")
    $BIN init "validator-$i" --chain-id "$CHAIN_ID" --home "$home" --default-denom "$DENOM" >/dev/null 2>&1
    # keys stay in the node's test keyring only; mnemonics are never written elsewhere
    $BIN keys add "validator-$i" --keyring-backend "$KEYRING" --home "$home" >/dev/null 2>&1
  done

  # Build validator-1's genesis as the canonical genesis file.
  local g1; g1="$(node_home 1)/config/genesis.json"
  local tmp; tmp=$(mktemp)
  jq --arg d "$DENOM" '
      .app_state.staking.params.bond_denom = $d
    | .app_state.crisis.constant_fee.denom = $d
    | .app_state.gov.params.min_deposit[0].denom = $d
    | .app_state.gov.params.expedited_min_deposit[0].denom = $d
    | .app_state.mint.params.mint_denom = $d
    | .app_state.gov.params.voting_period = "30s"
    | .app_state.gov.params.expedited_voting_period = "15s"
  ' "$g1" > "$tmp" && mv "$tmp" "$g1"

  for i in $(seq 1 $NUM_VALIDATORS); do
    local addr; addr=$($BIN keys show "validator-$i" -a --keyring-backend "$KEYRING" --home "$(node_home "$i")")
    $BIN genesis add-genesis-account "$addr" "$VAL_BALANCE" --home "$(node_home 1)"
  done

  # Each validator signs its own gentx against the shared genesis.
  mkdir -p "$NET_DIR/gentx"
  for i in $(seq 1 $NUM_VALIDATORS); do
    local home; home=$(node_home "$i")
    [[ $i -eq 1 ]] || cp "$g1" "$home/config/genesis.json"
    $BIN genesis gentx "validator-$i" "$VAL_STAKE" \
      --chain-id "$CHAIN_ID" --keyring-backend "$KEYRING" --home "$home" \
      --moniker "validator-$i" --output-document "$NET_DIR/gentx/gentx-$i.json" >/dev/null 2>&1
  done
  cp "$NET_DIR"/gentx/*.json "$(node_home 1)/config/gentx/" 2>/dev/null || {
    mkdir -p "$(node_home 1)/config/gentx" && cp "$NET_DIR"/gentx/*.json "$(node_home 1)/config/gentx/"
  }
  $BIN genesis collect-gentxs --home "$(node_home 1)" >/dev/null 2>&1
  $BIN genesis validate-genesis --home "$(node_home 1)"

  # Distribute the final genesis and each other validator's account keys to every node,
  # then wire ports and persistent peers.
  local peers=""
  for i in $(seq 1 $NUM_VALIDATORS); do
    local id; id=$($BIN comet show-node-id --home "$(node_home "$i")")
    peers+="${peers:+,}$id@127.0.0.1:$(p2p_port "$i")"
  done

  for i in $(seq 1 $NUM_VALIDATORS); do
    local home cfg app
    home=$(node_home "$i"); cfg="$home/config/config.toml"; app="$home/config/app.toml"
    [[ $i -eq 1 ]] || cp "$(node_home 1)/config/genesis.json" "$home/config/genesis.json"

    python3 "$ROOT_DIR/scripts/configure_node.py" \
      --config "$cfg" --app "$app" \
      --rpc-port "$(rpc_port "$i")" --p2p-port "$(p2p_port "$i")" \
      --pprof-port "$(pprof_port "$i")" --grpc-port "$(grpc_port "$i")" \
      --api-port "$(api_port "$i")" --peers "$peers" --denom "$DENOM"
  done

  echo "==> localnet initialized"
}

start_node() {
  local i=$1 home; home=$(node_home "$i")
  mkdir -p "$NET_DIR/logs"
  nohup "$BIN" start --home "$home" \
    --p2p.laddr "tcp://0.0.0.0:$(p2p_port "$i")" \
    --rpc.laddr "tcp://0.0.0.0:$(rpc_port "$i")" \
    > "$NET_DIR/logs/validator-$i.log" 2>&1 &
  echo $! > "$NET_DIR/validator-$i.pid"
  echo "==> validator-$i started (pid $(cat "$NET_DIR/validator-$i.pid"), rpc $(rpc_port "$i"))"
}

stop_node() {
  local i=$1 pidfile="$NET_DIR/validator-$1.pid"
  if [[ -f "$pidfile" ]]; then
    local pid; pid=$(cat "$pidfile")
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      for _ in $(seq 1 30); do kill -0 "$pid" 2>/dev/null || break; sleep 0.5; done
      kill -9 "$pid" 2>/dev/null || true
    fi
    rm -f "$pidfile"
    echo "==> validator-$i stopped"
  fi
}

wait_for_height() {
  local target=${1:-1} deadline=$((SECONDS + ${2:-120}))
  while (( SECONDS < deadline )); do
    local h
    h=$(curl -s "http://127.0.0.1:$(rpc_port 1)/status" | jq -r '.result.sync_info.latest_block_height // "0"' 2>/dev/null || echo 0)
    if [[ "$h" =~ ^[0-9]+$ ]] && (( h >= target )); then echo "$h"; return 0; fi
    sleep 1
  done
  return 1
}

cmd_start() {
  require_bin
  [[ -d "$(node_home 1)" ]] || init_network
  for i in $(seq 1 $NUM_VALIDATORS); do start_node "$i"; done
  echo "==> waiting for block production..."
  if h=$(wait_for_height 3 180); then
    echo "==> localnet is producing blocks (height $h)"
  else
    echo "!! localnet failed to produce blocks; see $NET_DIR/logs" >&2
    exit 1
  fi
  cmd_status
}

cmd_stop() { for i in $(seq 1 $NUM_VALIDATORS); do stop_node "$i"; done; }

cmd_clean() { cmd_stop; rm -rf "$NET_DIR"; echo "==> localnet state removed"; }

cmd_status() {
  for i in $(seq 1 $NUM_VALIDATORS); do
    local s; s=$(curl -s "http://127.0.0.1:$(rpc_port "$i")/status" || true)
    if [[ -z "$s" ]]; then
      echo "validator-$i: DOWN (rpc $(rpc_port "$i"))"
    else
      echo "validator-$i: height=$(jq -r '.result.sync_info.latest_block_height' <<<"$s")" \
           "catching_up=$(jq -r '.result.sync_info.catching_up' <<<"$s")" \
           "voting_power=$(jq -r '.result.validator_info.voting_power' <<<"$s")"
    fi
  done
}

case "${1:-start}" in
  start)   cmd_start ;;
  stop)    cmd_stop ;;
  clean)   cmd_clean ;;
  status)  cmd_status ;;
  init)    require_bin; init_network ;;
  restart-node) stop_node "$2"; sleep 2; start_node "$2" ;;
  *) echo "usage: $0 {start|stop|clean|status|init|restart-node <n>}" >&2; exit 1 ;;
esac
