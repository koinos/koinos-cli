package internal

import (
	"errors"
)

var (
	// EmptyCommandNameis the error returned when a command name is empty.
	ErrEmptyCommandName = errors.New("empty command name")

	//ErrUnknownCommand is the error returned when a command name is not found.
	ErrUnknownCommand = errors.New("unknown command")
)
