package cliutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestSaveWalletWindowsDoesNotInheritSharedAccess(t *testing.T) {
	// Removing the explicit creation-time ACL must expose the permissive parent
	// permissions and fail this test. Only disposable test keys are written.
	directory := t.TempDir()
	shared, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := shared.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(directory, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(directory, "private wallet 日本.wallet")
	key := bytes.Repeat([]byte{7}, 32)
	if err := SaveWalletFile(filename, "disposable-password", key); err != nil {
		t.Fatal(err)
	}
	security, err := windows.GetNamedSecurityInfo(filename, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := security.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatalf("wallet inherits permissions from its shared directory: control=%x, error=%v", control, err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	owner, _, err := security.Owner()
	if err != nil || owner == nil || !owner.Equals(user.User.Sid) {
		t.Fatalf("wallet is not owned by its creator: %v", err)
	}
	dacl, _, err = security.DACL()
	if err != nil || dacl == nil || dacl.AceCount != 1 {
		t.Fatalf("wallet does not have exactly one explicit access rule: %v", err)
	}
	var entry *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &entry); err != nil {
		t.Fatal(err)
	}
	grantee := (*windows.SID)(unsafe.Pointer(&entry.SidStart))
	if entry.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || entry.Header.AceFlags != 0 || !grantee.Equals(user.User.Sid) {
		t.Fatal("wallet grants access to someone other than its creator")
	}
	file, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	got, err := ReadWalletFile(file, "disposable-password")
	if err != nil || !bytes.Equal(got, key) {
		t.Fatalf("creator cannot reopen their wallet: %v", err)
	}
}
