package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/koinos/koinos-cli/internal/cli"
)

const extraToken = "register_token extra 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL EXTRA 8\n"

func startupTestEnvironment(t *testing.T) (string, *cli.ExecutionEnvironment) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	return dir, cli.NewExecutionEnvironment(nil, cli.NewCommandParser(cli.NewKoinosCommandSet()))
}

func TestStartupRequiresApprovalBeforeExecution(t *testing.T) {
	for _, decision := range []string{"approve", "decline", "cancel"} {
		t.Run(decision, func(t *testing.T) {
			dir, env := startupTestEnvironment(t)
			file := filepath.Join(dir, ".koinosrc")
			if err := os.WriteFile(file, []byte(extraToken), 0600); err != nil {
				t.Fatal(err)
			}
			reviews := 0
			review := func(path string, contents []byte) (bool, error) {
				reviews++
				if path != file || string(contents) != extraToken || env.Contracts.Contains("extra") {
					t.Fatal("preview was incomplete or commands executed before review")
				}
				if decision == "cancel" {
					return false, errors.New("cancelled")
				}
				return decision == "approve", nil
			}
			err := executeStartupFiles(dir, env.Parser, env, false, review)
			if (err != nil) != (decision == "cancel") {
				t.Fatalf("wrong cancellation result: %v", err)
			}
			if env.Contracts.Contains("extra") != (decision == "approve") {
				t.Fatal("startup did not respect the review decision")
			}
			if reviews != 1 {
				t.Fatalf("same home/local file reviewed %d times", reviews)
			}
		})
	}
}

func TestStartupExecutesReviewedBytesIfFileChanges(t *testing.T) {
	dir, env := startupTestEnvironment(t)
	file := filepath.Join(dir, ".koinosrc")
	if err := os.WriteFile(file, []byte(extraToken), 0600); err != nil {
		t.Fatal(err)
	}
	err := executeStartupFiles(dir, env.Parser, env, false, func(path string, contents []byte) (bool, error) {
		if string(contents) != extraToken {
			t.Fatal("wrong preview")
		}
		return true, os.WriteFile(path, []byte("create unseen.wallet test-password\n"), 0600)
	})
	if err != nil || !env.Contracts.Contains("extra") {
		t.Fatalf("reviewed contents did not execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "unseen.wallet")); !os.IsNotExist(err) {
		t.Fatal("executed unseen file contents written during review")
	}
}

func TestReviewedConfigKeepsExplicitNetworkAndAddsCommands(t *testing.T) {
	dir, env := startupTestEnvironment(t)
	server := startupChainServer(t, "EiAIKVvm6-V2qmsmUvPJy09vCCLbtn9lHFpwrJbcTIEWRQ==")
	defer server.Close()
	if err := env.Connect(context.Background(), server.URL); err != nil {
		t.Fatal(err)
	}
	// Only the verified network's config is eligible, and it cannot change nodes.
	if err := os.WriteFile(filepath.Join(dir, ".koinosrc"), []byte("unknown_mainnet_command\n"), 0600); err != nil {
		t.Fatal(err)
	}
	config := "disconnect; connect invalid-url\nregister_token koin 1FaSvLjQJsCJKq5ybmGsMMQs8RQYyVv8ju\n" + extraToken
	if err := os.WriteFile(filepath.Join(dir, ".koinosrc.testnet"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	err := executeStartupFiles(dir, env.Parser, env, true, func(_ string, _ []byte) (bool, error) { return true, nil })
	if err != nil {
		t.Fatal(err)
	}
	if !env.IsOnline() || env.RPCClient.Endpoint() != server.URL || env.NetworkLabel() != "testnet" {
		t.Fatal("reviewed config changed the explicitly selected network")
	}
	if result := cli.ParseAndInterpret(env.Parser, env, "help extra.balance_of; help koin.balance_of"); result.Err != nil {
		t.Fatal(result.Err)
	}
}

func TestEachStartupFileNeedsItsOwnApproval(t *testing.T) {
	dir, env := startupTestEnvironment(t)
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".koinosrc"), []byte(extraToken), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".koinosrc"), []byte(strings.ReplaceAll(extraToken, "extra", "local")), 0600); err != nil {
		t.Fatal(err)
	}
	reviews := 0
	err := executeStartupFiles(home, env.Parser, env, false, func(file string, _ []byte) (bool, error) {
		reviews++
		return file == filepath.Join(home, ".koinosrc"), nil
	})
	if err != nil || reviews != 2 || !env.Contracts.Contains("extra") || env.Contracts.Contains("local") {
		t.Fatalf("approval applied to a different file: reviews=%d err=%v", reviews, err)
	}
}

func TestApprovedStartupFailureStopsLaterFiles(t *testing.T) {
	dir, env := startupTestEnvironment(t)
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".koinosrc"), []byte("address\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".koinosrc"), []byte(extraToken), 0600); err != nil {
		t.Fatal(err)
	}
	reviews := 0
	err := executeStartupFiles(home, env.Parser, env, false, func(_ string, _ []byte) (bool, error) {
		reviews++
		return true, nil
	})
	if err == nil || reviews != 1 || env.Contracts.Contains("extra") {
		t.Fatalf("execution continued after failed startup: reviews=%d err=%v", reviews, err)
	}
}

func TestSymlinkedStartupFileIsReviewedOnlyOnce(t *testing.T) {
	dir, env := startupTestEnvironment(t)
	home := t.TempDir()
	file := filepath.Join(home, ".koinosrc")
	if err := os.WriteFile(file, []byte(extraToken), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(file, filepath.Join(dir, ".koinosrc")); err != nil {
		t.Fatal(err)
	}
	reviews := 0
	err := executeStartupFiles(home, env.Parser, env, false, func(_ string, _ []byte) (bool, error) {
		reviews++
		return true, nil
	})
	if err != nil || reviews != 1 || !env.Contracts.Contains("extra") {
		t.Fatalf("symlink duplicated review or execution: reviews=%d err=%v", reviews, err)
	}
}
