//go:build !windows

package cliutil

import "os"

func createWalletFile(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
}
