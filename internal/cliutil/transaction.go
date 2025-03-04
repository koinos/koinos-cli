package cliutil

import (
	"bytes"
	"context"
	"crypto/sha256"

	"github.com/btcsuite/btcd/btcec"
	"github.com/koinos/koinos-proto-golang/v2/koinos/canonical"
	"github.com/koinos/koinos-proto-golang/v2/koinos/protocol"
	util "github.com/koinos/koinos-util-golang/v2"
	"github.com/multiformats/go-multihash"
)

// CreateSignedTransaction creates a signed transaction
func CreateSignedTransaction(ctx context.Context, ops []*protocol.Operation, key *util.KoinosKey, nonce uint64, rcLimit uint64, chainID []byte, payer []byte) (*protocol.Transaction, error) {
	// Create the transaction
	transaction, err := CreateTransaction(ctx, ops, key.AddressBytes(), nonce, rcLimit, chainID, payer)
	if err != nil {
		return nil, err
	}

	// Sign the transaction
	err = SignTransaction(key.PrivateBytes(), transaction)
	if err != nil {
		return nil, err
	}

	return transaction, nil
}

// CreateTransaction creates a transaction from a list of operations with a specified payer
func CreateTransaction(ctx context.Context, ops []*protocol.Operation, address []byte, nonce uint64, rcLimit uint64, chainID []byte, payer []byte) (*protocol.Transaction, error) {
	var err error

	// Convert nonce to bytes
	nonceBytes, err := util.UInt64ToNonceBytes(nonce)
	if err != nil {
		return nil, err
	}

	// Get operation multihashes
	opHashes := make([][]byte, len(ops))
	for i, op := range ops {
		opHashes[i], err = util.HashMessage(op)
		if err != nil {
			return nil, err
		}
	}

	// Find merkle root
	merkleRoot, err := util.CalculateMerkleRoot(opHashes)
	if err != nil {
		return nil, err
	}

	// Create the header
	var header protocol.TransactionHeader
	if bytes.Equal(payer, address) {
		header = protocol.TransactionHeader{ChainId: chainID, RcLimit: rcLimit, Nonce: nonceBytes, OperationMerkleRoot: merkleRoot, Payer: payer}
	} else {
		header = protocol.TransactionHeader{ChainId: chainID, RcLimit: rcLimit, Nonce: nonceBytes, OperationMerkleRoot: merkleRoot, Payer: payer, Payee: address}
	}

	headerBytes, err := canonical.Marshal(&header)
	if err != nil {
		return nil, err
	}

	// Calculate the transaction ID
	sha256Hasher := sha256.New()
	sha256Hasher.Write(headerBytes)
	tid, err := multihash.Encode(sha256Hasher.Sum(nil), multihash.SHA2_256)
	if err != nil {
		return nil, err
	}

	// Create the transaction
	transaction := protocol.Transaction{Header: &header, Operations: ops, Id: tid}

	return &transaction, nil
}

// SignTransaction signs the transaction with the given key
func SignTransaction(key []byte, tx *protocol.Transaction) error {
	privateKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), key)

	// Decode the mutlihashed ID
	idBytes, err := multihash.Decode(tx.Id)
	if err != nil {
		return err
	}

	// Sign the transaction ID
	signatureBytes, err := btcec.SignCompact(btcec.S256(), privateKey, idBytes.Digest, true)
	if err != nil {
		return err
	}

	// Attach the signature data to the transaction
	if tx.Signatures == nil {
		tx.Signatures = [][]byte{signatureBytes}
	} else {
		tx.Signatures = append(tx.Signatures, signatureBytes)
	}

	return nil
}

// SignTransactionId signs the transaction ID with the given key
func SignTransactionId(key []byte, tid []byte) ([]byte, error) {
	privateKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), key)

	// Decode to multihash ID
	idBytes, err := multihash.Decode(tid)
	if err != nil {
		return nil, err
	}

	return btcec.SignCompact(btcec.S256(), privateKey, idBytes.Digest, true)
}
