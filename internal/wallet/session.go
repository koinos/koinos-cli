package wallet

import (
	"errors"

	"github.com/koinos/koinos-proto-golang/koinos/protocol"
)

var (
	ErrNoSession = errors.New("no session in progress")

	ErrSesionInProgress = errors.New("session in progress")
)

type OperationRequest struct {
	Op         *protocol.Operation
	LogMessage string
}

type TransactionSession struct {
	ops []OperationRequest
}

func (ts *TransactionSession) BeginSession() error {
	if ts.ops != nil {
		return ErrSesionInProgress
	}

	ts.ops = make([]OperationRequest, 0)
	return nil
}

func (ts *TransactionSession) EndSession() error {
	if ts.ops == nil {
		return ErrNoSession
	}

	ts.ops = nil
	return nil
}

func (ts *TransactionSession) AddOperation(op *protocol.Operation, logMessage string) error {
	if ts.ops == nil {
		return ErrNoSession
	}

	ts.ops = append(ts.ops, OperationRequest{Op: op, LogMessage: logMessage})
	return nil
}

func (ts *TransactionSession) GetOperations() ([]OperationRequest, error) {
	if ts.ops == nil {
		return nil, ErrNoSession
	}

	return ts.ops, nil
}

func (ts *TransactionSession) IsValid() bool {
	return ts.ops != nil
}
