//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	chainID  = "pinchain-local-1"
	denom    = "upin"
	fees     = "5000upin"
	gasLimit = "400000"
	node1    = "tcp://127.0.0.1:26657"
	node2    = "tcp://127.0.0.1:26667"
)

// repoRoot returns the repository root (tests/e2e/../..).
func repoRoot(t testing.TB) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func binPath(t testing.TB) string { return filepath.Join(repoRoot(t), "build", "pinchaind") }
func homeDir(t testing.TB) string { return filepath.Join(repoRoot(t), "localnet", "validator-1") }
func scriptPath(t testing.TB) string {
	return filepath.Join(repoRoot(t), "scripts", "localnet.sh")
}

// runCLI executes pinchaind against validator-1's home directory.
func runCLI(t testing.TB, args ...string) (string, error) {
	t.Helper()
	full := append([]string{"--home", homeDir(t)}, args...)
	cmd := exec.Command(binPath(t), full...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// mustRunCLI fails the test if the command exits non-zero.
func mustRunCLI(t testing.TB, args ...string) string {
	t.Helper()
	out, err := runCLI(t, args...)
	require.NoError(t, err, "pinchaind %s\n%s", strings.Join(args, " "), out)
	return out
}

// localnet drives scripts/localnet.sh.
func localnet(t testing.TB, args ...string) string {
	t.Helper()
	out, err := runLocalnet(args...)
	require.NoError(t, err, "localnet.sh %s\n%s", strings.Join(args, " "), out)
	return out
}

// runLocalnet drives scripts/localnet.sh without a *testing.T, for use in TestMain.
func runLocalnet(args ...string) (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root = filepath.Clean(filepath.Join(root, "..", ".."))

	cmd := exec.Command(filepath.Join(root, "scripts", "localnet.sh"), args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func txFlags() []string {
	return []string{
		"--chain-id", chainID,
		"--node", node1,
		"--keyring-backend", "test",
		"--fees", fees,
		"--gas", gasLimit,
		"--broadcast-mode", "sync",
		"--output", "json",
		"-y",
	}
}

func queryFlags() []string {
	return []string{"--node", node1, "--output", "json"}
}

// txResponse is the subset of the SDK tx response the tests assert on.
type txResponse struct {
	Code   uint32 `json:"code"`
	TxHash string `json:"txhash"`
	RawLog string `json:"raw_log"`
	Height string `json:"height"`
}

// parseTxResponse extracts the JSON tx response from CLI output.
func parseTxResponse(t testing.TB, out string) txResponse {
	t.Helper()
	idx := strings.Index(out, "{")
	require.GreaterOrEqual(t, idx, 0, "no JSON in output: %s", out)

	var res txResponse
	require.NoError(t, json.Unmarshal([]byte(out[idx:]), &res), out)
	return res
}

// broadcast submits a tx and waits for it to be included in a block, returning
// the committed tx result.
func broadcast(t testing.TB, args ...string) txResponse {
	t.Helper()
	out := mustRunCLI(t, args...)
	res := parseTxResponse(t, out)
	require.Equal(t, uint32(0), res.Code, "CheckTx failed: %s", out)
	require.NotEmpty(t, res.TxHash)
	return waitForTx(t, res.TxHash)
}

// waitForTx polls until the tx is committed and returns its result.
func waitForTx(t testing.TB, hash string) txResponse {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		out, err := runCLI(t, append([]string{"query", "tx", hash}, queryFlags()...)...)
		if err == nil && strings.Contains(out, "\"txhash\"") {
			res := parseTxResponse(t, out)
			if res.Height != "" && res.Height != "0" {
				return res
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("tx %s was not committed in time", hash)
	return txResponse{}
}

// createKey creates a new test-keyring account and returns its address.
func createKey(t testing.TB, name string) string {
	t.Helper()
	// Recreate deterministically for repeatable runs.
	_, _ = runCLI(t, "keys", "delete", name, "--keyring-backend", "test", "-y")
	mustRunCLI(t, "keys", "add", name, "--keyring-backend", "test")
	return keyAddress(t, name)
}

func keyAddress(t testing.TB, name string) string {
	t.Helper()
	out := mustRunCLI(t, "keys", "show", name, "-a", "--keyring-backend", "test")
	return strings.TrimSpace(out)
}

// balance returns the account balance in base units.
func balance(t testing.TB, addr string) int64 {
	t.Helper()
	out := mustRunCLI(t, append([]string{"query", "bank", "balances", addr}, queryFlags()...)...)

	var res struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}
	idx := strings.Index(out, "{")
	require.GreaterOrEqual(t, idx, 0, out)
	require.NoError(t, json.Unmarshal([]byte(out[idx:]), &res), out)

	for _, coin := range res.Balances {
		if coin.Denom == denom {
			amount, err := strconv.ParseInt(coin.Amount, 10, 64)
			require.NoError(t, err)
			return amount
		}
	}
	return 0
}

// pinRecord mirrors the JSON shape of a QueryPINResponse record.
type pinRecord struct {
	Pin            string `json:"pin"`
	Owner          string `json:"owner"`
	CreationHeight string `json:"creation_height"`
	Status         string `json:"status"`
}

// resolvePIN queries the chain for a PIN record.
func resolvePIN(t testing.TB, pin string) (pinRecord, error) {
	t.Helper()
	out, err := runCLI(t, append([]string{"query", "pin", "resolve", pin}, queryFlags()...)...)
	if err != nil {
		return pinRecord{}, fmt.Errorf("%s: %w", out, err)
	}

	var res struct {
		Record pinRecord `json:"record"`
	}
	idx := strings.Index(out, "{")
	if idx < 0 {
		return pinRecord{}, fmt.Errorf("no JSON in output: %s", out)
	}
	if err := json.Unmarshal([]byte(out[idx:]), &res); err != nil {
		return pinRecord{}, fmt.Errorf("%s: %w", out, err)
	}
	return res.Record, nil
}

// nodeStatus returns the sync info of a node RPC endpoint.
func nodeStatus(t testing.TB, nodeURL string) (height int64, catchingUp bool, err error) {
	t.Helper()
	out, err := runCLI(t, "status", "--node", nodeURL)
	if err != nil {
		return 0, false, fmt.Errorf("%s: %w", out, err)
	}

	var res struct {
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
			CatchingUp        bool   `json:"catching_up"`
		} `json:"sync_info"`
	}
	idx := strings.Index(out, "{")
	if idx < 0 {
		return 0, false, fmt.Errorf("no JSON in status output: %s", out)
	}
	if err := json.Unmarshal([]byte(out[idx:]), &res); err != nil {
		return 0, false, fmt.Errorf("%s: %w", out, err)
	}
	height, err = strconv.ParseInt(res.SyncInfo.LatestBlockHeight, 10, 64)
	return height, res.SyncInfo.CatchingUp, err
}

// waitForHeight blocks until the node reports at least the target height.
func waitForHeight(t testing.TB, nodeURL string, target int64, timeout time.Duration) int64 {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		height, catchingUp, err := nodeStatus(t, nodeURL)
		if err == nil && !catchingUp && height >= target {
			return height
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("node %s did not reach height %d in %s", nodeURL, target, timeout)
	return 0
}
