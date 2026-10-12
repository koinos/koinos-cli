package cli

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func assertWalletOwnerOnly(t *testing.T, filename string) {
	t.Helper()
	security, err := windows.GetNamedSecurityInfo(filename, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := security.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatalf("wallet inherits directory access rules: %v", err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	owner, _, err := security.Owner()
	if err != nil || owner == nil || !owner.Equals(user.User.Sid) {
		t.Fatalf("wallet is not owned by its creator: %v", err)
	}
	dacl, _, err := security.DACL()
	if err != nil || dacl == nil || dacl.AceCount != 1 {
		t.Fatalf("wallet must have exactly one access rule: %v", err)
	}
	var entry *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &entry); err != nil {
		t.Fatal(err)
	}
	grantee := (*windows.SID)(unsafe.Pointer(&entry.SidStart))
	if entry.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || entry.Header.AceFlags != 0 || !grantee.Equals(user.User.Sid) {
		t.Fatal("wallet grants access beyond its creator")
	}
}
