//go:build unix

package cliutil

import (
	"bytes"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
)

func TestFailedWalletWriteRemovesCreatedFile(t *testing.T) {
	if os.Getenv("KOINOS_CLI_TEST_WRITE_FAILURE") == "1" {
		// Simulate a full filesystem without changing the parent process's limits.
		signal.Ignore(syscall.SIGXFSZ)
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: 0, Max: 0}); err != nil {
			t.Fatal(err)
		}
		filename := os.Getenv("KOINOS_CLI_TEST_WALLET_PATH")
		if err := SaveWalletFile(filename, "test-password", bytes.Repeat([]byte{1}, 32)); err == nil {
			t.Fatal("write unexpectedly succeeded")
		}
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			t.Fatalf("failed write left file behind: %v", err)
		}
		return
	}
	filename := filepath.Join(t.TempDir(), "failed.wallet")
	command := exec.Command(os.Args[0], "-test.run=^TestFailedWalletWriteRemovesCreatedFile$")
	command.Env = append(os.Environ(), "KOINOS_CLI_TEST_WRITE_FAILURE=1", "KOINOS_CLI_TEST_WALLET_PATH="+filename)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("write failure probe: %v\n%s", err, output)
	}
}
