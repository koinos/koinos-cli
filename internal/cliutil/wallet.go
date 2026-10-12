package cliutil

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/minio/sio"
	"golang.org/x/crypto/argon2"
)

// Version 1 has an eight-byte magic/version, a 32-byte random salt, then a
// DARE v2 ciphertext. Its fixed Argon2id parameters avoid trusting file-supplied
// memory or CPU costs. Changing the derivation requires a new format version.
const walletMagic = "KOINOSW\x01"
const walletSaltSize = 32

func walletConfig(key []byte) sio.Config {
	return sio.Config{
		MinVersion:   sio.Version20,
		MaxVersion:   sio.Version20,
		CipherSuites: []byte{sio.AES_GCM, sio.CHACHA20_POLY1305},
		Key:          key,
	}
}

func walletKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, 3, 64*1024, 4, 32)
}

// CreateWalletFile writes a new-format encrypted wallet to an open file.
// Call SaveWalletFile when creating a wallet by filename.
func CreateWalletFile(file *os.File, passphrase string, privateKey []byte) error {
	if passphrase == "" {
		return ErrEmptyPassphrase
	}
	salt := make([]byte, walletSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	key := walletKey(passphrase, salt)
	defer clear(key)
	header := append([]byte(walletMagic), salt...)
	if _, err := file.Write(header); err != nil {
		return err
	}
	_, err := sio.Encrypt(file, bytes.NewReader(privateKey), walletConfig(key))
	return err
}

// SaveWalletFile creates an owner-only wallet without replacing an existing
// file (including a symlink). Failed writes remove only the file we created.
func SaveWalletFile(filename, passphrase string, privateKey []byte) (err error) {
	if passphrase == "" {
		return ErrEmptyPassphrase
	}
	file, err := createWalletFile(filename)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: %s", ErrWalletExists, filename)
		}
		return err
	}
	defer func() {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(filename)
		}
	}()
	if err = CreateWalletFile(file, passphrase, privateKey); err != nil {
		return err
	}
	return file.Sync()
}

// ReadWalletFile reads both legacy SHA-256 wallets and the versioned format.
// Reading never migrates or otherwise rewrites an existing wallet.
func ReadWalletFile(file *os.File, passphrase string) ([]byte, error) {
	if passphrase == "" {
		return nil, ErrEmptyPassphrase
	}
	reader := bufio.NewReader(file)
	header, err := reader.Peek(len(walletMagic))
	if err != nil {
		return nil, err
	}
	var key []byte
	if string(header) == walletMagic {
		if _, err := reader.Discard(len(walletMagic)); err != nil {
			return nil, err
		}
		salt := make([]byte, walletSaltSize)
		if _, err := io.ReadFull(reader, salt); err != nil {
			return nil, err
		}
		key = walletKey(passphrase, salt)
	} else {
		if bytes.HasPrefix(header, []byte("KOINOSW")) {
			return nil, fmt.Errorf("unsupported wallet format version")
		}
		hash := sha256.Sum256([]byte(passphrase))
		key = hash[:]
	}
	defer clear(key)
	var destination bytes.Buffer
	if _, err := sio.Decrypt(&destination, reader, walletConfig(key)); err != nil {
		clear(destination.Bytes())
		return nil, err
	}
	return destination.Bytes(), nil
}
