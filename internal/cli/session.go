package cli

import (
	"errors"

	"github.com/koinos/koinos-proto-golang/koinos/protocol"
)

var (
	// ErrNoSession no session is in progress
	ErrNoSession = errors.New("no session in progress")

	// ErrSesionInProgress session is in progress
	ErrSesionInProgress = errors.New("session in progress")
)

// PendingOperation is an operation in a TransactionSession
type PendingOperation struct {
	Op         *protocol.Operation
	LogMessage string
}

// TransactionSession allows for adding multiple operations to a single transaction
type TransactionSession struct {
	ops []PendingOperation
}

// BeginSession if none is in progress
func (ts *TransactionSession) BeginSession() error {
	if ts.ops != nil {
		return ErrSesionInProgress
	}

	ts.ops = make([]PendingOperation, 0)
	return nil
}

// EndSession if one is in progress
func (ts *TransactionSession) EndSession() error {
	if ts.ops == nil {
		return ErrNoSession
	}

	ts.ops = nil
	return nil
}

// AddOperation to session
func (ts *TransactionSession) AddOperation(op *protocol.Operation, logMessage string) error {
	if ts.ops == nil {
		return ErrNoSession
	}

	ts.ops = append(ts.ops, PendingOperation{Op: op, LogMessage: logMessage})
	return nil
}

// GetOperations in current session
func (ts *TransactionSession) GetOperations() ([]PendingOperation, error) {
	if ts.ops == nil {
		return nil, ErrNoSession
	}

	return ts.ops, nil
}

// IsValid if session is in progress
func (ts *TransactionSession) IsValid() bool {
	return ts.ops != nil
}
