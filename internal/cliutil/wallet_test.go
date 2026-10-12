package cliutil

import (
	"bytes"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/minio/sio"
	"golang.org/x/crypto/argon2"
)

func walletTestFile(t *testing.T, data []byte) *os.File {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "wallet")
	if err := os.WriteFile(filename, data, 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filename, os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestNewWalletUsesSaltedPasswordDerivation(t *testing.T) {
	privateKey := bytes.Repeat([]byte{1}, 32)
	f := walletTestFile(t, nil)
	if err := CreateWalletFile(f, "test-password", privateKey); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 40 || string(data[:8]) != "KOINOSW\x01" {
		t.Fatal("wallet lacks versioned password-derivation header")
	}
	// Decode the published format independently of the wallet reader.
	key := argon2.IDKey([]byte("test-password"), data[8:40], 3, 64*1024, 4, 32)
	var plain bytes.Buffer
	_, err = sio.Decrypt(&plain, bytes.NewReader(data[40:]), sio.Config{Key: key})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain.Bytes(), privateKey) {
		t.Fatal("unexpected private key")
	}
	second := walletTestFile(t, nil)
	if err := CreateWalletFile(second, "test-password", privateKey); err != nil {
		t.Fatal(err)
	}
	second.Seek(0, io.SeekStart)
	other, _ := io.ReadAll(second)
	if bytes.Equal(data[8:40], other[8:40]) {
		t.Fatal("wallets reused a salt")
	}
}

func TestLegacyWalletRemainsReadableAndUnchanged(t *testing.T) {
	privateKey := bytes.Repeat([]byte{2}, 32)
	// Produced with the pre-maintenance sio v0.3.0 writer and a fixed test nonce.
	legacy, err := hex.DecodeString("20001f00c2424242424242424242424233b3ee3372935c982196a30b4adfcf2f1cbccd94d88bc8afbdaa19a9354378a7eb82c76e91416283c9f41c2668c770cb")
	if err != nil {
		t.Fatal(err)
	}
	f := walletTestFile(t, legacy)
	got, err := ReadWalletFile(f, "legacy-password")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, privateKey) {
		t.Fatal("legacy key changed")
	}
	f.Seek(0, io.SeekStart)
	after, _ := io.ReadAll(f)
	if !bytes.Equal(after, legacy) {
		t.Fatal("opening rewrote legacy wallet")
	}
}

func TestWalletRejectsWrongPasswordAndDamage(t *testing.T) {
	f := walletTestFile(t, nil)
	if err := CreateWalletFile(f, "correct-password", bytes.Repeat([]byte{3}, 32)); err != nil {
		t.Fatal(err)
	}
	f.Seek(0, io.SeekStart)
	data, _ := io.ReadAll(f)
	for _, tc := range []struct {
		name     string
		password string
		mutate   func([]byte) []byte
	}{
		{"wrong password", "wrong", func(b []byte) []byte { return b }},
		{"truncated", "correct-password", func(b []byte) []byte { return b[:len(b)-1] }},
		{"changed ciphertext", "correct-password", func(b []byte) []byte { b[len(b)-1] ^= 1; return b }},
		{"changed header", "correct-password", func(b []byte) []byte { b[7] ^= 1; return b }},
		{"changed salt", "correct-password", func(b []byte) []byte { b[10] ^= 1; return b }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.mutate(append([]byte(nil), data...))
			got, err := ReadWalletFile(walletTestFile(t, input), tc.password)
			if err == nil || len(got) != 0 {
				t.Fatalf("damaged wallet exposed plaintext: error=%v", err)
			}
		})
	}
}

func TestSaveWalletRejectsDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	filename := filepath.Join(dir, "wallet")
	if err := os.Symlink(target, filename); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := SaveWalletFile(filename, "password", bytes.Repeat([]byte{1}, 32)); err == nil {
		t.Fatal("followed dangling wallet symlink")
	}
	st, err := os.Lstat(filename)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink replaced")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("created symlink target")
	}
}
