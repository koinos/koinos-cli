package cliutil

import (
	"errors"
)

var (
	// ErrInvalidCommandName is the error returned when a command name is empty.
	ErrInvalidCommandName = errors.New("invalid command name")

	// ErrUnknownCommand is the error returned when a command name is not found.
	ErrUnknownCommand = errors.New("unknown command")

	// ErrNotEnoughArguments is returned when a command was not passed enough arguments.
	ErrNotEnoughArguments = errors.New("not enough arguments")

	// ErrMissingParam is returned when a parameter is missing.
	ErrMissingParam = errors.New("missing parameter")

	// ErrInvalidParam is returned when a parameter is invalid.
	ErrInvalidParam = errors.New("invalid value given for parameter")

	// ErrInvalidResponse is returned when a response from the RPC endpoint is invalid
	ErrInvalidResponse = errors.New("invalid response")

	// ErrUnexpectedHashLength is returned when the passphrase hash length is incorrect
	ErrUnexpectedHashLength = errors.New("unexpected hash length")

	// ErrEmptyPassphrase is returned when the user supplies an empty passphrase
	ErrEmptyPassphrase = errors.New("passphrase cannot be empty")

	// ErrWalletExists is returned when trying to create a new wallet and it already exists
	ErrWalletExists = errors.New("wallet already exists")

	// ErrWalletClosed is returned when an open wallet is needed, but no wallet is open
	ErrWalletClosed = errors.New("no open wallet")

	// ErrWalletDecrypt is returned when a wallet file does not decrypt properly
	ErrWalletDecrypt = errors.New("wallet decryption failed")

	// ErrInvalidPrivateKey is returned when an imported private key is invalid
	ErrInvalidPrivateKey = errors.New("invalid private key")

	// ErrInvalidAmount is returned when an amount is invalid
	ErrInvalidAmount = errors.New("invalid amount")

	// ErrOffline is returned when a wallet is not online
	ErrOffline = errors.New("wallet is offline")

	// ErrFileNotFound is returned when the file is not found
	ErrFileNotFound = errors.New("file not found")

	// ErrBlankPassword is returned when the user supplies a blank password
	ErrBlankPassword = errors.New("blank password")

	// ErrInvalidABI is returned when an ABI is invalid
	ErrInvalidABI = errors.New("invalid ABI")

	// ErrUnsupportedType is returned when an unsupported type is passed
	ErrUnsupportedType = errors.New("unsupported type")

	// ErrContract is returned when a contract is already registered
	ErrContract = errors.New("contract error")

	// ErrInsufficientRC is returned when not enough resource credits can be used to cover a transaction
	ErrInsufficientRC = errors.New("insufficient rc")
)
