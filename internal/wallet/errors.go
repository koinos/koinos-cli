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

	// ErrEmptyParam is returned when a parameter is empty.
	ErrEmptyParam = errors.New("empty parameter")
)
