//go:build e2e

package e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSecurity exercises the adversarial cases against the live localnet. It
// relies on the accounts and PINs created by TestPINEndToEnd.
func TestSecurity(t *testing.T) {
	alice := keyAddress(t, "alice")
	bob := keyAddress(t, "bob")

	t.Run("malformed pin is rejected client side", func(t *testing.T) {
		for _, pin := range []string{"A7K92XM", "A7K-92X", "A7K-92XMM", "A7K-92X!", ""} {
			out, err := runCLI(t, append([]string{"tx", "pin", "register", pin, "--from", "alice"}, txFlags()...)...)
			require.Error(t, err, "pin %q was accepted: %s", pin, out)
			require.Contains(t, out, "invalid pin")
		}
	})

	t.Run("excluded characters are rejected", func(t *testing.T) {
		for _, pin := range []string{"A7I-92XM", "A7O-92XM", "A7L-92XM", "A70-92XM", "A71-92XM", "B4M-81QZ"} {
			out, err := runCLI(t, append([]string{"tx", "pin", "register", pin, "--from", "alice"}, txFlags()...)...)
			require.Error(t, err, "pin %q was accepted: %s", pin, out)
			require.Contains(t, out, "invalid pin")
		}
	})

	t.Run("duplicate pin registration fails on chain", func(t *testing.T) {
		res := broadcastExpectFailure(t, append([]string{
			"tx", "pin", "register", alicePIN, "--from", "bob",
		}, txFlags()...)...)
		require.Contains(t, res.RawLog, "already registered")

		// Ownership is unchanged: the PIN still resolves to Alice.
		record, err := resolvePIN(t, alicePIN)
		require.NoError(t, err)
		require.Equal(t, alice, record.Owner)
	})

	t.Run("lowercase pin resolves to the canonical record", func(t *testing.T) {
		upper, err := resolvePIN(t, alicePIN)
		require.NoError(t, err)
		lower, err := resolvePIN(t, strings.ToLower(alicePIN))
		require.NoError(t, err)
		require.Equal(t, upper, lower)
		require.Equal(t, alicePIN, lower.Pin)
	})

	t.Run("nonexistent pin cannot be resolved or paid", func(t *testing.T) {
		_, err := resolvePIN(t, "Z9Z-9999")
		require.Error(t, err)

		res := broadcastExpectFailure(t, append([]string{
			"tx", "pin", "send-by-pin", "Z9Z-9999", "1PIN", "--from", "alice",
		}, txFlags()...)...)
		require.Contains(t, res.RawLog, "pin not found")
	})

	t.Run("zero and negative amounts are rejected", func(t *testing.T) {
		for _, amount := range []string{"0PIN", "0upin", "-1PIN", "-1upin"} {
			out, err := runCLI(t, append([]string{
				"tx", "pin", "send-by-pin", bobPIN, amount, "--from", "alice",
			}, txFlags()...)...)
			require.Error(t, err, "amount %q was accepted: %s", amount, out)
		}
	})

	t.Run("insufficient balance cannot move funds", func(t *testing.T) {
		poor := createKey(t, "poor")
		fund(t, "validator-1", poor, 50_000) // enough for fees only
		bobBefore := balance(t, bob)

		res := broadcastExpectFailure(t, append([]string{
			"tx", "pin", "send-by-pin", bobPIN, "1000PIN", "--from", "poor",
		}, txFlags()...)...)
		require.Contains(t, res.RawLog, "insufficient funds")
		require.Equal(t, bobBefore, balance(t, bob))
	})

	t.Run("overflow boundary amount fails without moving funds", func(t *testing.T) {
		bobBefore := balance(t, bob)
		huge := "115792089237316195423570985008687907853269984665640564039457584007913129639935upin"
		out, err := runCLI(t, append([]string{
			"tx", "pin", "send-by-pin", bobPIN, huge, "--from", "alice",
		}, txFlags()...)...)
		if err == nil {
			res := parseTxResponse(t, out)
			if res.Code == 0 && res.TxHash != "" {
				committed := waitForTx(t, res.TxHash)
				require.NotEqual(t, uint32(0), committed.Code, "huge transfer must not succeed")
			}
		}
		require.Equal(t, bobBefore, balance(t, bob))
	})

	t.Run("tampered signature is rejected", func(t *testing.T) {
		dir := t.TempDir()
		unsigned := filepath.Join(dir, "unsigned.json")
		signed := filepath.Join(dir, "signed.json")

		out := mustRunCLI(t, "tx", "pin", "send-by-pin", bobPIN, "1PIN",
			"--from", "alice", "--chain-id", chainID, "--node", node1,
			"--keyring-backend", "test", "--fees", fees, "--gas", gasLimit,
			"--generate-only")
		require.NoError(t, os.WriteFile(unsigned, []byte(out), 0o600))

		mustRunCLI(t, "tx", "sign", unsigned, "--from", "alice",
			"--chain-id", chainID, "--node", node1, "--keyring-backend", "test",
			"--output-document", signed)

		tampered := filepath.Join(dir, "tampered.json")
		require.NoError(t, os.WriteFile(tampered, []byte(flipSignature(t, signed)), 0o600))

		out, cliErr := runCLI(t, "tx", "broadcast", tampered, "--node", node1, "--output", "json")
		if cliErr == nil {
			res := parseTxResponse(t, out)
			require.NotEqual(t, uint32(0), res.Code, "tampered signature was accepted: %s", out)
		}
		require.Contains(t, strings.ToLower(out), "signature")
	})

	t.Run("replayed transaction is rejected", func(t *testing.T) {
		dir := t.TempDir()
		unsigned := filepath.Join(dir, "unsigned.json")
		signed := filepath.Join(dir, "signed.json")

		out := mustRunCLI(t, "tx", "pin", "send-by-pin", bobPIN, "1PIN",
			"--from", "alice", "--chain-id", chainID, "--node", node1,
			"--keyring-backend", "test", "--fees", fees, "--gas", gasLimit,
			"--generate-only")
		require.NoError(t, os.WriteFile(unsigned, []byte(out), 0o600))
		mustRunCLI(t, "tx", "sign", unsigned, "--from", "alice",
			"--chain-id", chainID, "--node", node1, "--keyring-backend", "test",
			"--output-document", signed)

		bobBefore := balance(t, bob)

		first := parseTxResponse(t, mustRunCLI(t, "tx", "broadcast", signed,
			"--node", node1, "--broadcast-mode", "sync", "--output", "json"))
		require.Equal(t, uint32(0), first.Code, first.RawLog)
		committed := waitForTx(t, first.TxHash)
		require.Equal(t, uint32(0), committed.Code, committed.RawLog)
		require.Equal(t, bobBefore+1_000_000, balance(t, bob))

		// Rebroadcasting the exact same signed bytes must not transfer again.
		replay, cliErr := runCLI(t, "tx", "broadcast", signed,
			"--node", node1, "--broadcast-mode", "sync", "--output", "json")
		if cliErr == nil {
			res := parseTxResponse(t, replay)
			require.NotEqual(t, uint32(0), res.Code, "replay was accepted: %s", replay)
		}
		require.Equal(t, bobBefore+1_000_000, balance(t, bob))
	})

	t.Run("malformed transaction body is rejected", func(t *testing.T) {
		dir := t.TempDir()
		garbage := filepath.Join(dir, "garbage.json")
		require.NoError(t, os.WriteFile(garbage, []byte(`{"body":"not-a-tx"}`), 0o600))

		out, cliErr := runCLI(t, "tx", "broadcast", garbage, "--node", node1, "--output", "json")
		require.Error(t, cliErr, "malformed tx was accepted: %s", out)
	})

	t.Run("pin ownership cannot be taken over", func(t *testing.T) {
		// Bob cannot re-register Alice's PIN, and Alice's PIN keeps pointing at
		// Alice, so funds sent to it never reach Bob.
		aliceRecord, err := resolvePIN(t, alicePIN)
		require.NoError(t, err)
		require.Equal(t, alice, aliceRecord.Owner)

		bobBefore := balance(t, bob)
		aliceBefore := balance(t, alice)
		res := broadcast(t, append([]string{
			"tx", "pin", "send-by-pin", alicePIN, "1PIN", "--from", "alice",
		}, txFlags()...)...)
		require.Equal(t, uint32(0), res.Code, res.RawLog)
		require.Equal(t, bobBefore, balance(t, bob))
		// Alice paid only the fee: the transfer went back to herself.
		require.Equal(t, aliceBefore-5000, balance(t, alice))
	})
}

// broadcastExpectFailure submits a tx that is expected to fail in DeliverTx and
// returns the committed result.
func broadcastExpectFailure(t testing.TB, args ...string) txResponse {
	t.Helper()
	out, cliErr := runCLI(t, args...)
	require.NoError(t, cliErr, out)

	res := parseTxResponse(t, out)
	if res.Code != 0 {
		return res
	}
	committed := waitForTx(t, res.TxHash)
	require.NotEqual(t, uint32(0), committed.Code, "tx unexpectedly succeeded: %s", out)
	return committed
}

// flipSignature returns the signed tx JSON with its signature bytes corrupted.
func flipSignature(t testing.TB, path string) string {
	t.Helper()
	raw, readErr := os.ReadFile(path)
	require.NoError(t, readErr)

	var tx map[string]any
	require.NoError(t, json.Unmarshal(raw, &tx))

	sigs, ok := tx["signatures"].([]any)
	require.True(t, ok, "no signatures in %s", raw)
	require.NotEmpty(t, sigs)

	sig, ok := sigs[0].(string)
	require.True(t, ok)
	// Corrupt the first base64 character deterministically.
	if sig[0] == 'A' {
		sig = "B" + sig[1:]
	} else {
		sig = "A" + sig[1:]
	}
	sigs[0] = sig
	tx["signatures"] = sigs

	tampered, marshalErr := json.Marshal(tx)
	require.NoError(t, marshalErr)
	return string(tampered)
}
