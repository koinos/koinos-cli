package wallet

import (
	"crypto/ecdsa"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
)

// KoinosKey represents a set of keys
type KoinosKey struct {
	PrivateKey *ecdsa.PrivateKey
}

// GenerateKoinosKey generates a new set of keys
func GenerateKoinosKey() (*KoinosKey, error) {
	k, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	keys := &KoinosKey{PrivateKey: k}
	return keys, nil
}

// NewKoinosKeysFromBytes creates a new key set from a private key byte slice
func NewKoinosKeysFromBytes(private []byte) (*KoinosKey, error) {
	pk, err := crypto.ToECDSA(private)
	if err != nil {
		return nil, err
	}

	return &KoinosKey{PrivateKey: pk}, nil
}

// Address displays the base58 address associated with this key set
func (keys *KoinosKey) Address() string {
	_, pubkey := btcec.PrivKeyFromBytes(btcec.S256(), keys.PrivateBytes())
	mainNetAddr, _ := btcutil.NewAddressPubKey(pubkey.SerializeCompressed(), &chaincfg.MainNetParams)
	return mainNetAddr.EncodeAddress()
}

// Private gets the private key in base58
func (keys *KoinosKey) Private() string {
	return EncodeWIF(crypto.FromECDSA(keys.PrivateKey), false, 128)
}

// Public gets the public key in base58
func (keys *KoinosKey) Public() string {
	return base58.Encode(crypto.FromECDSAPub(&keys.PrivateKey.PublicKey))
}

// PublicBytes get the public key bytes
func (keys *KoinosKey) PublicBytes() []byte {
	return crypto.FromECDSAPub(&keys.PrivateKey.PublicKey)
}

// PrivateBytes gets the private key bytes
func (keys *KoinosKey) PrivateBytes() []byte {
	return crypto.FromECDSA(keys.PrivateKey)
}
