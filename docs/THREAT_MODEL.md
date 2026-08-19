# Threat model

Scope: the PINChain node (`pinchaind`), the `x/pin` module and the local
four-validator network as they exist today. Wallet, explorer, indexer, faucet and
public networks do not exist yet and are out of scope.

## Assets

1. Account private keys (held by users, never by the chain).
2. Token balances in `x/bank`.
3. PIN records (the `PIN -> address` mapping) in consensus state.
4. Consensus liveness and safety of the validator set.

## Trust assumptions

- Cosmos SDK v0.50 signature verification, replay protection and fee deduction are
  trusted and are the only authorization path used by `x/pin`.
- CometBFT BFT assumptions hold: safety with < 1/3 byzantine voting power.
- The localnet keyring uses the `test` backend and is for development only.

## Properties enforced today

| Property | Mechanism | Test |
| --- | --- | --- |
| Only the key holder can register a PIN for their account | `cosmos.msg.v1.signer` + ante handler signature check | `tests/e2e/security_test.go` (tampered signature) |
| A PIN cannot be hijacked or reassigned | duplicate registration rejected; no transfer/revoke message exists | `TestRegisterPINRejectsDuplicates`, E2E "pin ownership cannot be taken over" |
| Case/whitespace variants cannot create a second record for the same PIN | normalization in `ValidateBasic` and in the keeper before every read/write | `TestNormalizePIN`, E2E lowercase resolution |
| Malformed PINs never enter state | strict alphabet + length validation, rejected client-side and on-chain | `TestValidatePIN`, `TestRegisterPINRejectsMalformed` |
| Transfers move real bank funds only, with no shortcut | handler calls `x/bank.SendCoins`; module holds no balances and has no module account | `TestSendByPIN`, E2E exact-balance assertions |
| No transfer to an unknown or revoked PIN | `ResolvePIN` requires an existing `ACTIVE` record | `TestResolvePINRejectsRevoked`, E2E nonexistent PIN |
| Zero, negative, duplicated or malformed-denom amounts are rejected | `Coins.Validate()` + `IsAllPositive()` in `ValidateBasic` and again in the handler | `TestMsgSendByPINValidateBasic`, E2E zero/negative |
| Overspending is impossible | bank keeper insufficient-funds error; huge amounts fail the same way | `TestSendByPINOverflowBoundary`, E2E overflow boundary |
| Transfers to blocked module addresses are refused | `BlockedAddr` check before sending | `TestSendByPINFailures` |
| Replay of a signed transaction fails | SDK account sequence numbers | E2E "replayed transaction is rejected" |
| Fees are always charged for a state-changing attempt | standard ante handler; localnet enforces `--minimum-gas-prices 0.001upin` | E2E balance deltas |
| PIN generation is unpredictable | `crypto/rand` only; no timestamps, counters or heights | `TestGeneratePIN` |
| Validator restart does not fork or lose state | CometBFT WAL/state sync; restart is covered end to end | E2E validator-2 restart |

## Key handling

- The chain never sees or stores private keys, mnemonics or seed phrases.
- A PIN grants no authority: it is public data. Learning a PIN allows sending
  funds *to* its owner and nothing else.
- Localnet validator keys are generated inside each node's `test` keyring under
  `localnet/`, which is git-ignored. Scripts never print mnemonics, and no key
  material is committed.

## Known risks and open items

- **PIN squatting / enumeration.** Registration is first-come, first-served and the
  keyspace (31^7 ≈ 2.75e10) is enumerable by a funded adversary. Only the
  transaction fee limits bulk registration; no per-PIN price, rate limit or
  reservation policy exists yet.
- **PIN typos are unrecoverable.** A valid-but-wrong PIN resolves to a different
  real account, and the transfer succeeds. Wallet UX must show the resolved address
  before signing. Excluding `0/1/I/O/L` reduces but does not remove this risk.
- **No revocation path.** `PIN_STATUS_REVOKED` is honoured by resolution, but no
  message can set it, so a compromised account's PIN cannot be retired yet.
- **No key rotation / recovery for PINs.** A PIN is permanently bound to the
  registering address.
- **`test` keyring backend on the localnet.** Unencrypted by design; never use it
  for a network holding value.
- **Open RPC/API on localnet.** Ports bind to `0.0.0.0` with permissive CORS for
  development convenience; public deployments must front them with authentication
  and rate limiting.
- **Unaudited.** No external security review, fuzzing or load testing has been
  performed (Phase 8).
- **Gov/staking/slashing parameters are defaults** and have not been tuned or
  tested against attacks (Phase 4).
- **IBC is scaffolded but untested** in this repository.
