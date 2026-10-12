//go:build !windows

package cli

import (
	"os"
	"testing"
)

func assertWalletOwnerOnly(t *testing.T, filename string) {
	t.Helper()
	st, err := os.Stat(filename)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0077 != 0 {
		t.Fatalf("wallet file permissions are %04o; readable outside owner", st.Mode().Perm())
	}
}
