package cliutil

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/koinos/koinos-proto-golang/v2/koinos/protocol"
	util "github.com/koinos/koinos-util-golang/v2"
)

// Version is set by release builds with -ldflags -X. Unversioned local builds
// must not identify themselves as an older published release.
var Version = "development"

// Hardcoded Koin contract constants
const (
	KoinSymbol         = "KOIN"
	ManaSymbol         = "mana"
	KoinPrecision      = 8
	KoinContractID     = "19GYjDBVXU7keLbYvMLazsGQn3GTWHjHkK"
	KoinBalanceOfEntry = uint32(0x5c721497)
	KoinTransferEntry  = uint32(0x27f576ca)
)

// Hardcoded Multihash constants.
const (
	RIPEMD128 = 0x1052
	RIPEMD160 = 0x1053
	RIPEMD256 = 0x1054
	RIPEMD320 = 0x1055
)

// TransactionReceiptToString creates a string from a receipt
func TransactionReceiptToString(receipt *protocol.TransactionReceipt, operations int) string {
	s := fmt.Sprintf("Transaction with ID 0x%s containing %d operations", hex.EncodeToString(receipt.Id), operations)
	if receipt.Reverted {
		s += " reverted."
	} else {
		s += " submitted."
	}

	// Build the mana result
	manaDec, err := util.SatoshiToDecimal(receipt.RcUsed, KoinPrecision)
	if err != nil {
		s += "\n" + err.Error()
		return s
	}

	s += fmt.Sprintf("\nMana cost: %v (Disk: %d, Network: %d, Compute: %d)", manaDec, receipt.DiskStorageUsed, receipt.NetworkBandwidthUsed, receipt.ComputeBandwidthUsed)

	// Show logs if available
	if len(receipt.Logs) > 0 {
		s += "\nLogs:"
		for _, log := range receipt.Logs {
			s += "\n" + log
		}
	}

	return s
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
