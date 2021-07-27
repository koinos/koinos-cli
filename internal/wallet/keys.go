package wallet

import (
	"crypto/ecdsa"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
)

// KoinosKeys represents a set of keys
type KoinosKeys struct {
	PrivateKey *ecdsa.PrivateKey
}

// GenerateKoinosKeys generates a new set of keys
func GenerateKoinosKeys() (*KoinosKeys, error) {
	k, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	keys := &KoinosKeys{PrivateKey: k}
	return keys, nil
}

// NewKoinosKeysFromBytes creates a new key set from a private key byte slice
func NewKoinosKeysFromBytes(private []byte) (*KoinosKeys, error) {
	pk, err := crypto.ToECDSA(private)
	if err != nil {
		return nil, err
	}

	return &KoinosKeys{PrivateKey: pk}, nil
}

// Address displays the base58 address associated with this key set
func (keys *KoinosKeys) Address() string {
	return base58.Encode(crypto.PubkeyToAddress(keys.PrivateKey.PublicKey).Bytes())
}

// Private gets the private key in base58
func (keys *KoinosKeys) Private() string {
	return base58.Encode(crypto.FromECDSA(keys.PrivateKey))
}

// Public gets the public key in base58
func (keys *KoinosKeys) Public() string {
	return base58.Encode(crypto.FromECDSAPub(&keys.PrivateKey.PublicKey))
}
