package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/koinos/koinos-cli/internal/cliutil"
)

func TestCreatePromptsTwiceAndOpenPromptsOnce(t *testing.T) {
	t.Setenv("WALLET_PASS", "")
	filename := filepath.Join(t.TempDir(), "prompt.wallet")
	ee := NewExecutionEnvironment(nil, nil)
	prompts := 0
	ee.PasswordPrompt = func(label string) (string, error) { prompts++; return "disposable-secret", nil }
	if _, err := (&CreateCommand{Filename: filename}).Execute(context.Background(), ee); err != nil {
		t.Fatal(err)
	}
	if prompts != 2 || !ee.IsWalletOpen() {
		t.Fatalf("creation should confirm password; prompts=%d", prompts)
	}
	address := string(ee.Key.AddressBytes())
	ee.CloseWallet()
	if _, err := (&OpenCommand{Filename: filename}).Execute(context.Background(), ee); err != nil {
		t.Fatal(err)
	}
	if prompts != 3 || string(ee.Key.AddressBytes()) != address {
		t.Fatal("opening did not recover the created wallet with one prompt")
	}
}

func TestPromptMismatchBlankAndCancellationLeaveNoWallet(t *testing.T) {
	for _, mode := range []string{"mismatch", "blank", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("WALLET_PASS", "")
			filename := filepath.Join(t.TempDir(), "rejected.wallet")
			ee := NewExecutionEnvironment(nil, nil)
			calls := 0
			ee.PasswordPrompt = func(label string) (string, error) {
				calls++
				if mode == "cancel" {
					return "", errors.New("cancelled")
				}
				if mode == "blank" {
					return "", nil
				}
				if calls == 1 {
					return "first-secret", nil
				}
				return "second-secret", nil
			}
			_, err := (&CreateCommand{Filename: filename}).Execute(context.Background(), ee)
			if err == nil {
				t.Fatal("invalid prompted password accepted")
			}
			if calls == 0 {
				t.Fatal("password prompt was never used")
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("error revealed a password")
			}
			if _, err := os.Stat(filename); !os.IsNotExist(err) {
				t.Fatal("rejected prompt created a wallet")
			}
		})
	}
}

func TestExplicitAndEnvironmentPasswordsRemainNoninteractive(t *testing.T) {
	for _, explicit := range []bool{true, false} {
		t.Run(map[bool]string{true: "argument", false: "environment"}[explicit], func(t *testing.T) {
			t.Setenv("WALLET_PASS", "environment-secret")
			ee := NewExecutionEnvironment(nil, nil)
			ee.PasswordPrompt = func(string) (string, error) { t.Fatal("explicit credentials triggered a prompt"); return "", nil }
			secret := "argument-secret"
			var password *string
			if explicit {
				password = &secret
			} else {
				secret = "environment-secret"
			}
			filename := filepath.Join(t.TempDir(), "script.wallet")
			if _, err := (&CreateCommand{Filename: filename, Password: password}).Execute(context.Background(), ee); err != nil {
				t.Fatal(err)
			}
			file, err := os.Open(filename)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			if _, err := cliutil.ReadWalletFile(file, secret); err != nil {
				t.Fatal("wallet did not use the specified credential", err)
			}
		})
	}
}

func TestCreatedWalletIsOwnerOnly(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "test.wallet")
	password := "disposable-test-password"
	cmd := &CreateCommand{Filename: filename, Password: &password}
	if _, err := cmd.Execute(context.Background(), NewExecutionEnvironment(nil, nil)); err != nil {
		t.Fatal(err)
	}
	assertWalletOwnerOnly(t, filename)
}
func TestRejectedPasswordLeavesNoWallet(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "test.wallet")
	password := ""
	cmd := &CreateCommand{Filename: filename, Password: &password}
	if _, err := cmd.Execute(context.Background(), NewExecutionEnvironment(nil, nil)); err == nil {
		t.Fatal("expected password error")
	}
	if st, err := os.Stat(filename); err == nil {
		t.Fatalf("failed creation left a wallet file of %d bytes", st.Size())
	}
}

func TestCreateDoesNotOverwriteWallet(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "existing.wallet")
	original := []byte("existing wallet contents")
	if err := os.WriteFile(filename, original, 0600); err != nil {
		t.Fatal(err)
	}
	password := "test-password"
	_, err := (&CreateCommand{Filename: filename, Password: &password}).Execute(context.Background(), NewExecutionEnvironment(nil, nil))
	if err == nil {
		t.Fatal("expected existing wallet error")
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatal("existing wallet was changed")
	}
}
