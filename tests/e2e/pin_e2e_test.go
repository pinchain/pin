//go:build e2e

package e2e_test

import (
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	alicePIN = "A7K-92XM"
	// The brief's example PIN for Bob (B4M-81QZ) contains "1", which the PIN
	// alphabet excludes, so the E2E flow uses the nearest valid PIN.
	bobPIN = "B4M-89QZ"

	fundAlice = int64(100_000_000) // 100 PIN
	fundBob   = int64(10_000_000)  // 10 PIN
	sendPIN   = "25PIN"
	sendBase  = int64(25_000_000)
)

// TestMain brings up the 4-validator localnet used by every test in this package.
func TestMain(m *testing.M) {
	for _, args := range [][]string{{"clean"}, {"start"}} {
		if out, err := runLocalnet(args...); err != nil {
			log.Fatalf("localnet %v failed: %v\n%s", args, err, out)
		}
	}

	code := m.Run()

	if os.Getenv("KEEP_LOCALNET") == "" {
		if out, err := runLocalnet("stop"); err != nil {
			log.Printf("localnet stop failed: %v\n%s", err, out)
		}
	}
	os.Exit(code)
}

// TestPINEndToEnd runs the full milestone flow: real blocks, PIN registration,
// PIN resolution, a real bank transfer by PIN and validator restart recovery.
func TestPINEndToEnd(t *testing.T) {
	// 1. localnet is producing blocks on all four validators.
	for _, node := range []string{node1, node2, "tcp://127.0.0.1:26677", "tcp://127.0.0.1:26687"} {
		height := waitForHeight(t, node, 2, 90*time.Second)
		t.Logf("%s at height %d", node, height)
	}

	// 2/3. Create Alice and Bob.
	alice := createKey(t, "alice")
	bob := createKey(t, "bob")
	require.NotEqual(t, alice, bob)
	require.True(t, strings.HasPrefix(alice, "pin1"), alice)

	// 4. Fund both accounts from a genesis validator account.
	fund(t, "validator-1", alice, fundAlice)
	fund(t, "validator-1", bob, fundBob)
	require.Equal(t, fundAlice, balance(t, alice))
	require.Equal(t, fundBob, balance(t, bob))

	// 5/6. Register both PINs with real signed transactions.
	registerPIN(t, "alice", alicePIN)
	registerPIN(t, "bob", bobPIN)

	// 7/8. Bob's PIN resolves to Bob's address on chain.
	record, err := resolvePIN(t, bobPIN)
	require.NoError(t, err)
	require.Equal(t, bobPIN, record.Pin)
	require.Equal(t, bob, record.Owner)
	require.Equal(t, "PIN_STATUS_ACTIVE", record.Status)
	require.NotEqual(t, "0", record.CreationHeight)

	// Lowercase input normalizes to the same record.
	lower, err := resolvePIN(t, strings.ToLower(bobPIN))
	require.NoError(t, err)
	require.Equal(t, record, lower)

	aliceBefore := balance(t, alice)
	bobBefore := balance(t, bob)

	// 9/10. Alice sends 25 PIN to Bob's PIN and the tx is committed.
	res := broadcast(t, append([]string{
		"tx", "pin", "send-by-pin", strings.ToLower(bobPIN), sendPIN, "--from", "alice",
	}, txFlags()...)...)
	require.Equal(t, uint32(0), res.Code, res.RawLog)

	// 11/12. Bob received exactly 25 PIN; Alice paid exactly 25 PIN plus fees.
	require.Equal(t, bobBefore+sendBase, balance(t, bob))
	require.Equal(t, aliceBefore-sendBase-5000, balance(t, alice))

	// 13/14. The transaction exists in a real block.
	require.NotEmpty(t, res.Height)
	require.NotEqual(t, "0", res.Height)
	require.Contains(t, res.RawLog+mustRunCLI(t, append([]string{"query", "tx", res.TxHash}, queryFlags()...)...), "send_by_pin")

	// 15/16. Restart validator-2 and verify it catches up with the network.
	heightBefore, _, err := nodeStatus(t, node1)
	require.NoError(t, err)
	localnet(t, "restart-node", "2")
	target := heightBefore + 3
	height := waitForHeight(t, node2, target, 120*time.Second)
	t.Logf("validator-2 caught up to height %d (target %d)", height, target)

	// The restarted validator serves the same PIN state.
	out := mustRunCLI(t, "query", "pin", "resolve", bobPIN, "--node", node2, "--output", "json")
	require.Contains(t, out, bob)
}

// fund transfers base-denom coins from a genesis account to addr.
func fund(t testing.TB, from, addr string, amount int64) {
	t.Helper()
	res := broadcast(t, append([]string{
		"tx", "bank", "send", from, addr, amountString(amount), "--from", from,
	}, txFlags()...)...)
	require.Equal(t, uint32(0), res.Code, res.RawLog)
}

// registerPIN registers pin for the named key.
func registerPIN(t testing.TB, key, pin string) {
	t.Helper()
	res := broadcast(t, append([]string{
		"tx", "pin", "register", pin, "--from", key,
	}, txFlags()...)...)
	require.Equal(t, uint32(0), res.Code, res.RawLog)
}

func amountString(amount int64) string {
	return strconv.FormatInt(amount, 10) + denom
}
