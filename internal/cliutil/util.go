package cliutil

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/minio/sio"
)

const (
	// Version number (this should probably not live here)
	Version = "v0.2.0"
)

func walletConfig(password []byte) sio.Config {
	return sio.Config{
		MinVersion:     sio.Version20,
		MaxVersion:     sio.Version20,
		CipherSuites:   []byte{sio.AES_256_GCM, sio.CHACHA20_POLY1305},
		Key:            password,
		SequenceNumber: uint32(0),
	}
}

// CreateWalletFile creates a new wallet file on disk
func CreateWalletFile(file *os.File, passphrase string, privateKey []byte) error {
	hasher := sha256.New()
	bytesWritten, err := hasher.Write([]byte(passphrase))

	if err != nil {
		return err
	}

	if bytesWritten <= 0 {
		return ErrEmptyPassphrase
	}

	passwordHash := hasher.Sum(nil)

	if len(passwordHash) != 32 {
		return ErrUnexpectedHashLength
	}

	source := bytes.NewReader(privateKey)
	_, err = sio.Encrypt(file, source, walletConfig(passwordHash))

	return err
}

// ReadWalletFile extracts the private key from the provided wallet file
func ReadWalletFile(file *os.File, passphrase string) ([]byte, error) {
	hasher := sha256.New()
	bytesWritten, err := hasher.Write([]byte(passphrase))

	if err != nil {
		return nil, err
	}

	if bytesWritten <= 0 {
		return nil, ErrEmptyPassphrase
	}

	passwordHash := hasher.Sum(nil)

	if len(passwordHash) != 32 {
		return nil, ErrUnexpectedHashLength
	}

	var destination bytes.Buffer
	_, err = sio.Decrypt(&destination, file, walletConfig(passwordHash))

	return destination.Bytes(), err
}

// GetPassword takes the password input from a command, and returns the string password which should be used
func GetPassword(password *string) (string, error) {
	// Get the password
	result := ""
	if password == nil { // If no password is provided, check the environment variable
		result = os.Getenv("WALLET_PASS")
		// Advise about the environment variable
		if result == "" {
			return result, fmt.Errorf("%w: no password was provided and env variable WALLET_PASS is empty", ErrBlankPassword)
		}
	} else {
		result = *password
	}

	// If the result is blank, return an error
	if result == "" {
		return result, fmt.Errorf("%w: password cannot be empty", ErrBlankPassword)
	}

	return result, nil
}
