package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCreatedWalletIsOwnerOnly(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "test.wallet")
	password := "disposable-test-password"
	cmd := &CreateCommand{Filename: filename, Password: &password}
	if _, err := cmd.Execute(context.Background(), NewExecutionEnvironment(nil, nil)); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(filename)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0077 != 0 {
		t.Fatalf("wallet file permissions are %04o; readable outside owner", st.Mode().Perm())
	}
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
