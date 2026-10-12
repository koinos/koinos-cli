package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
	command := exec.Command(os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
	command.Dir = dir
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
	code, output := invokeCLI(t, dir, "--file", script)
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

func TestSameStartupFileRunsOnce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".koinosrc"), []byte("create once.wallet test-password\n"), 0600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "commands")
	if err := os.WriteFile(script, []byte("address\n"), 0600); err != nil {
		t.Fatal(err)
	}
	code, output := invokeCLI(t, dir, "--file", script)
	if code != 0 {
		t.Fatalf("same home/local startup file ran twice: %s", output)
	}
}
