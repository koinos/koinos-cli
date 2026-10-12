package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExplicitRPCRequiresReviewForAutomaticConfiguration(t *testing.T) {
	for _, test := range []struct {
		name, chainID, config string
	}{
		{"mainnet", "EiBZK_GGVP0H_fXVAM3j6EAuz3-B-l3ejxRSewi7qIBfSA==", ".koinosrc"},
		{"testnet", "EiAIKVvm6-V2qmsmUvPJy09vCCLbtn9lHFpwrJbcTIEWRQ==", ".koinosrc.testnet"},
		{"custom", "AQ==", ".koinosrc"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if test.config != ".koinosrc" {
				if err := os.WriteFile(filepath.Join(dir, ".koinosrc"), []byte("unknown_mainnet_command\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			// Multiple commands on one line must retain the selected connection too.
			config := "disconnect; connect invalid-url; register_token example 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL EXAMPLE 8\n"
			if err := os.WriteFile(filepath.Join(dir, test.config), []byte(config), 0600); err != nil {
				t.Fatal(err)
			}
			server := startupChainServer(t, test.chainID)
			defer server.Close()
			code, output := invokeCLI(t, dir, "--rpc", server.URL, "--execute", "help example.balance_of; status")
			if code != 1 || !strings.Contains(output, "requires interactive review") || !strings.Contains(output, test.config) {
				t.Fatalf("automatic config did not require review: code=%d output=%s", code, output)
			}
		})
	}
}

func startupChainServer(t *testing.T, chainID string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Method != "chain.get_chain_id" {
			t.Errorf("unexpected RPC method %s", request.Method)
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"chain_id":%q}}`, request.ID, chainID)
	}))
}

func TestNoRCWithExplicitRPCSkipsStartup(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".koinosrc.testnet"), []byte("unknown_startup_command\n"), 0600); err != nil {
		t.Fatal(err)
	}
	server := startupChainServer(t, "EiAIKVvm6-V2qmsmUvPJy09vCCLbtn9lHFpwrJbcTIEWRQ==")
	defer server.Close()
	code, output := invokeCLI(t, dir, "--no-rc", "--rpc", server.URL, "--execute", "status")
	if code != 0 || !strings.Contains(output, "Network: testnet") {
		t.Fatalf("--no-rc did not skip startup: code=%d output=%s", code, output)
	}
}

func TestExplicitFileStillRunsConnectionCommands(t *testing.T) {
	dir := t.TempDir()
	initial := startupChainServer(t, "AQ==")
	defer initial.Close()
	next := startupChainServer(t, "Ag==")
	defer next.Close()
	script := "disconnect; connect " + next.URL + "; status\n"
	if err := os.WriteFile(filepath.Join(dir, "custom.config"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	code, output := invokeCLI(t, dir, "--no-rc", "--rpc", initial.URL, "--file", "custom.config")
	if code != 0 || !strings.Contains(output, "Endpoint: "+next.URL) {
		t.Fatalf("explicit config connection was ignored: code=%d output=%s", code, output)
	}
}

func TestMalformedStartupLineStopsBeforeEarlierCommands(t *testing.T) {
	dir := t.TempDir()
	config := "create should-not-exist.wallet test-password; unknown_command\n"
	if err := os.WriteFile(filepath.Join(dir, ".koinosrc"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	server := startupChainServer(t, "AQ==")
	defer server.Close()
	code, output := invokeCLI(t, dir, "--no-rc", "--rpc", server.URL, "--file", ".koinosrc")
	if code != 1 || !strings.Contains(output, ".koinosrc:1") {
		t.Fatalf("malformed startup line did not stop execution: code=%d output=%s", code, output)
	}
	if _, err := os.Stat(filepath.Join(dir, "should-not-exist.wallet")); !os.IsNotExist(err) {
		t.Fatal("partially parsed startup line created a wallet")
	}
}

func TestNetworkSelectionRejectsConflictingOrUnknownOptions(t *testing.T) {
	for _, args := range [][]string{{"--network", "wrong", "--execute", "list"}, {"--network", "testnet", "--rpc", "http://localhost", "--execute", "list"}} {
		code, output := invokeCLI(t, t.TempDir(), args...)
		if code != 2 || !strings.Contains(output, "network") {
			t.Fatalf("invalid network selection: code=%d output=%s", code, output)
		}
	}
}

// Execute the real entrypoint in an isolated process and configuration directory.
func TestCLIProcess(t *testing.T) {
	if os.Getenv("KOINOS_CLI_TEST_PROCESS") != "1" {
		return
	}
	start := 0
	for i, arg := range os.Args {
		if arg == "--" {
			start = i + 1
			break
		}
	}
	os.Exit(run(os.Args[start:], os.Getenv("KOINOS_CLI_TEST_CONFIG")))
}

func invokeCLI(t *testing.T, dir string, args ...string) (int, string) {
	t.Helper()
	return invokeCLIWithInput(t, dir, "", args...)
}

func invokeCLIWithInput(t *testing.T, dir, input string, args ...string) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
	command.Dir = dir
	command.Stdin = strings.NewReader(input)
	command.Env = append(os.Environ(), "KOINOS_CLI_TEST_PROCESS=1", "KOINOS_CLI_TEST_CONFIG="+dir)
	output, err := command.CombinedOutput()
	if err == nil {
		return 0, string(output)
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode(), string(output)
	}
	t.Fatal(err)
	return -1, ""
}

func TestNonterminalWalletCreationDoesNotPromptOrCreateFile(t *testing.T) {
	t.Setenv("WALLET_PASS", "")
	dir := t.TempDir()
	code, output := invokeCLI(t, dir, "--no-rc", "--execute", "create unattended.wallet")
	if code != 1 || !strings.Contains(output, "WALLET_PASS") || strings.Contains(output, "New wallet password:") {
		t.Fatalf("missing script credentials should fail without prompting: code=%d output=%s", code, output)
	}
	if _, err := os.Stat(filepath.Join(dir, "unattended.wallet")); !os.IsNotExist(err) {
		t.Fatal("missing script password left a wallet file")
	}
}

func TestBatchFailureExitStatus(t *testing.T) {
	for _, args := range [][]string{
		{"--execute", "unknown_command"},
		{"--execute", "address"},
		{"--file", "missing.commands"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			code, output := invokeCLI(t, t.TempDir(), args...)
			if code == 0 {
				t.Fatalf("failed batch exited successfully: %s", output)
			}
		})
	}
}

func TestBatchStopsBeforeLaterCommands(t *testing.T) {
	for _, file := range []bool{false, true} {
		t.Run(map[bool]string{false: "execute", true: "file"}[file], func(t *testing.T) {
			dir := t.TempDir()
			args := []string{"--execute", "address; create should-not-exist.wallet test-password"}
			if file {
				script := filepath.Join(dir, "commands")
				if err := os.WriteFile(script, []byte("address\ncreate should-not-exist.wallet test-password\n"), 0600); err != nil {
					t.Fatal(err)
				}
				args = []string{"--file", script}
			}
			code, output := invokeCLI(t, dir, args...)
			if code == 0 {
				t.Fatalf("failed batch exited successfully: %s", output)
			}
			if _, err := os.Stat(filepath.Join(dir, "should-not-exist.wallet")); !os.IsNotExist(err) {
				t.Fatal("command after failure created a wallet")
			}
		})
	}
}

func TestSuccessfulBatchAndHelpExitStatus(t *testing.T) {
	for _, args := range [][]string{{"--execute", "list"}, {"--help"}, {"--version"}} {
		code, output := invokeCLI(t, t.TempDir(), args...)
		if code != 0 {
			t.Fatalf("%v exited %d: %s", args, code, output)
		}
	}
}

func TestStartupCommandFailureStopsExplicitScript(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".koinosrc"), []byte("address\n"), 0600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "commands")
	if err := os.WriteFile(script, []byte("create should-not-exist.wallet test-password\n"), 0600); err != nil {
		t.Fatal(err)
	}
	code, output := invokeCLI(t, dir, "--no-rc", "--file", ".koinosrc", "--file", script)
	if code == 0 {
		t.Fatalf("startup failure exited successfully: %s", output)
	}
	if _, err := os.Stat(filepath.Join(dir, "should-not-exist.wallet")); !os.IsNotExist(err) {
		t.Fatal("explicit script ran after startup failure")
	}
}

func TestNoRCIsolatesBatchFromStartupCommands(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".koinosrc"), []byte("address\n"), 0600); err != nil {
		t.Fatal(err)
	}
	code, output := invokeCLI(t, dir, "--no-rc", "--execute", "list")
	if code != 0 {
		t.Fatalf("isolated batch ran startup commands: %s", output)
	}
}

func TestInvalidFlagsExplainFailure(t *testing.T) {
	for _, args := range [][]string{{"--unknown"}, {"--rpc"}} {
		code, output := invokeCLI(t, t.TempDir(), args...)
		if code == 0 || strings.TrimSpace(output) == "" {
			t.Fatalf("invalid flags must fail with a diagnostic; code=%d output=%q", code, output)
		}
	}
}

func TestStandardTokenCommandsNeedNoStartupFile(t *testing.T) {
	for _, chainID := range []string{
		"EiBZK_GGVP0H_fXVAM3j6EAuz3-B-l3ejxRSewi7qIBfSA==",
		"EiAIKVvm6-V2qmsmUvPJy09vCCLbtn9lHFpwrJbcTIEWRQ==",
	} {
		t.Run(chainID, func(t *testing.T) {
			dir := t.TempDir()
			server := startupChainServer(t, chainID)
			defer server.Close()
			code, output := invokeCLI(t, dir, "--no-rc", "--rpc", server.URL,
				"--execute", "help koin.balance_of; help koin.transfer; help vhp.total_supply")
			if code != 0 || !strings.Contains(output, "Usage: koin.balance_of") || !strings.Contains(output, "Usage: koin.transfer") || !strings.Contains(output, "Usage: vhp.total_supply") {
				t.Fatalf("standard commands required a startup file: code=%d output=%s", code, output)
			}
		})
	}
}

func TestConfigAddsCustomCommandsAlongsideStandardTokens(t *testing.T) {
	for _, configName := range []string{".koinosrc.testnet", "custom.config"} {
		t.Run(configName, func(t *testing.T) {
			dir := t.TempDir()
			// No arguments or return type means the contract's standard empty messages.
			abi := `{"methods":{"ping":{"entry-point":"0x12345678","read-only":true}}}`
			if err := os.WriteFile(filepath.Join(dir, "custom.abi"), []byte(abi), 0600); err != nil {
				t.Fatal(err)
			}
			config := "register_token koin 1FaSvLjQJsCJKq5ybmGsMMQs8RQYyVv8ju\n" +
				"register_token vhp 17n12ktwN79sR6ia9DDgCfmw77EgpbTyBi VHP 8\n" +
				"register_token custom 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL CUSTOM 4\n" +
				"register custom_contract 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL custom.abi\n" +
				"help custom.balance_of; help custom_contract.ping; help koin.transfer; help vhp.balance_of\n"
			if err := os.WriteFile(filepath.Join(dir, configName), []byte(config), 0600); err != nil {
				t.Fatal(err)
			}
			server := startupChainServer(t, "EiAIKVvm6-V2qmsmUvPJy09vCCLbtn9lHFpwrJbcTIEWRQ==")
			defer server.Close()
			args := []string{"--no-rc", "--rpc", server.URL, "--file", configName}
			code, output := invokeCLI(t, dir, args...)
			if code != 0 {
				t.Fatalf("config extensions failed: code=%d output=%s", code, output)
			}
			for _, name := range []string{"custom.balance_of", "custom_contract.ping", "koin.transfer", "vhp.balance_of"} {
				if !strings.Contains(output, "Usage: "+name) {
					t.Fatalf("config command %s missing: %s", name, output)
				}
			}
		})
	}
}

func TestUnreviewedStartupStopsBatchBeforeAnyCommands(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".koinosrc"), []byte("create automatic.wallet test-password\n"), 0600); err != nil {
		t.Fatal(err)
	}
	code, output := invokeCLI(t, dir, "--execute", "create explicit.wallet test-password")
	if code != 1 || !strings.Contains(output, "requires interactive review") || !strings.Contains(output, "--no-rc") {
		t.Fatalf("unreviewed startup was not blocked: code=%d output=%s", code, output)
	}
	for _, name := range []string{"automatic.wallet", "explicit.wallet"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("unreviewed startup ran a command: %s", name)
		}
	}
}

func TestPipedYesCannotApproveAutomaticStartup(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".koinosrc"), []byte("create automatic.wallet test-password\n"), 0600); err != nil {
		t.Fatal(err)
	}
	code, output := invokeCLIWithInput(t, dir, "yes\n", "--force-interactive", "--execute", "list")
	if code != 1 || !strings.Contains(output, "requires interactive review") {
		t.Fatalf("piped input bypassed terminal review: code=%d output=%s", code, output)
	}
	if _, err := os.Stat(filepath.Join(dir, "automatic.wallet")); !os.IsNotExist(err) {
		t.Fatal("piped input approved an automatic config")
	}
}
