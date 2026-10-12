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

func TestSaveWalletWindowsRejectsAlternateDataStreams(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "existing.wallet")
	original := []byte("existing wallet contents")
	if err := os.WriteFile(filename, original, 0600); err != nil {
		t.Fatal(err)
	}
	stream := filename + ":hidden"
	if err := SaveWalletFile(stream, "disposable-password", bytes.Repeat([]byte{7}, 32)); err == nil {
		t.Fatal("created a wallet inside an existing file's alternate data stream")
	}
	if _, err := os.Stat(stream); !os.IsNotExist(err) {
		t.Fatalf("rejected wallet left an alternate data stream: %v", err)
	}
	got, err := os.ReadFile(filename)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatalf("rejected wallet changed the existing file: %v", err)
	}
}

func TestWindowsWalletRejectsUnverifiedAccessRules(t *testing.T) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sid := user.User.Sid.String()
	for _, tc := range []struct {
		name string
		sddl string
		ok   bool
	}{
		{"private owner", "O:" + sid + "D:P(A;;FA;;;" + sid + ")", true},
		{"inherited access", "O:" + sid + "D:(A;;FA;;;" + sid + ")", false},
		{"missing owner", "D:P(A;;FA;;;" + sid + ")", false},
		{"different owner", "O:WDD:P(A;;FA;;;" + sid + ")", false},
		{"everyone access", "O:" + sid + "D:P(A;;FA;;;WD)", false},
		{"extra grantee", "O:" + sid + "D:P(A;;FA;;;" + sid + ")(A;;FR;;;WD)", false},
		{"empty access list", "O:" + sid + "D:P", false},
		{"deny rule", "O:" + sid + "D:P(D;;FA;;;" + sid + ")", false},
		{"inherit only rule", "O:" + sid + "D:P(A;IO;FA;;;" + sid + ")", false},
		{"read only owner", "O:" + sid + "D:P(A;;FR;;;" + sid + ")", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			security, err := windows.SecurityDescriptorFromString(tc.sddl)
			if err != nil {
				t.Fatal(err)
			}
			if err := verifyWalletSecurity(security, user.User.Sid); (err == nil) != tc.ok {
				t.Fatalf("accepted=%v, want %v: %v", err == nil, tc.ok, err)
			}
		})
	}
}
