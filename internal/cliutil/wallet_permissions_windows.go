package cliutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// createWalletFile applies the creator's protected ACL in CreateFile itself.
// Chmod on Windows only changes the read-only attribute, not access permissions.
func createWalletFile(filename string) (file *os.File, err error) {
	defer func() {
		if err != nil {
			err = &os.PathError{Op: "create private wallet", Path: filename, Err: err}
		}
	}()
	// Alternate streams share another file's security descriptor and are not
	// independent wallet files. IsLocal also rejects reserved device names.
	if !filepath.IsLocal(filepath.Base(filename)) || strings.Contains(filename[len(filepath.VolumeName(filename)):], ":") {
		return nil, errors.New("wallet must be an ordinary file, not a device or alternate data stream")
	}
	name, err := windows.UTF16PtrFromString(filename)
	if err != nil {
		return nil, err
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	// P disables inheritance; the sole access rule grants this user full access.
	// Set the owner explicitly, including when running with an elevated token.
	sid := user.User.Sid.String()
	security, err := windows.SecurityDescriptorFromString("O:" + sid + "D:P(A;;FA;;;" + sid + ")")
	if err != nil {
		return nil, err
	}
	attributes := windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: security,
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_WRITE|windows.READ_CONTROL|windows.DELETE, 0, &attributes, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			// Delete by handle while exclusive access still prevents replacement.
			// No wallet bytes have been written if verification fails here.
			remove := byte(1)
			deleteErr := windows.SetFileInformationByHandle(handle, windows.FileDispositionInfo, &remove, 1)
			err = errors.Join(err, deleteErr, windows.CloseHandle(handle))
		}
	}()
	var volumeFlags uint32
	if err := windows.GetVolumeInformationByHandle(handle, nil, 0, nil, nil, &volumeFlags, nil, 0); err != nil {
		return nil, fmt.Errorf("cannot verify filesystem access controls: %w", err)
	}
	if volumeFlags&windows.FILE_PERSISTENT_ACLS == 0 {
		return nil, errors.New("wallet creation requires a filesystem with Windows access controls, such as NTFS")
	}
	actual, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return nil, err
	}
	if err := verifyWalletSecurity(actual, user.User.Sid); err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), filename), nil
}

func verifyWalletSecurity(security *windows.SECURITY_DESCRIPTOR, user *windows.SID) error {
	control, _, err := security.Control()
	if err != nil {
		return err
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		return errors.New("wallet access rules are not protected from directory inheritance")
	}
	owner, _, err := security.Owner()
	if err != nil {
		return err
	}
	if owner == nil || !owner.Equals(user) {
		return errors.New("wallet owner is not the current user")
	}
	dacl, _, err := security.DACL()
	if err != nil {
		return err
	}
	if dacl == nil || dacl.AceCount != 1 {
		return errors.New("wallet access rules are not limited to its owner")
	}
	var entry *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &entry); err != nil {
		return err
	}
	grantee := (*windows.SID)(unsafe.Pointer(&entry.SidStart))
	required := windows.ACCESS_MASK(windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE | windows.DELETE)
	if entry.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || entry.Header.AceFlags != 0 || !grantee.Equals(user) || entry.Mask&required != required {
		return errors.New("wallet access rule does not grant private owner access")
	}
	return nil
}
