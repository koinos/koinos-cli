package wallet

import (
	"errors"
)

var (
	// ErrEmptyCommandName is the error returned when a command name is empty.
	ErrEmptyCommandName = errors.New("empty command name")

	// ErrUnknownCommand is the error returned when a command name is not found.
	ErrUnknownCommand = errors.New("unknown command")

	// ErrNotEnoughArguments is returned when a command was not passed enough arguments.
	ErrNotEnoughArguments = errors.New("not enough arguments")

	// ErrMissingParam is returned when a parameter is missing.
	ErrMissingParam = errors.New("missing parameter")

	// ErrInvalidString is returned when a string is invalid.
	ErrInvalidString = errors.New("invalid string")
)
