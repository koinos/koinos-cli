package wallet

import (
	"errors"

	"github.com/koinos/koinos-proto-golang/koinos/protocol"
)

var (
	ErrNoSession = errors.New("no session in progress")

	ErrSesionInProgress = errors.New("session in progress")
)

type TransactionSession struct {
	ops []*protocol.Operation
}

func (ts *TransactionSession) BeginSession() error {
	if ts.ops != nil {
		return ErrSesionInProgress
	}

	ts.ops = make([]*protocol.Operation, 0)
	return nil
}

func (ts *TransactionSession) EndSession() error {
	if ts.ops == nil {
		return ErrNoSession
	}

	ts.ops = nil
	return nil
}

func (ts *TransactionSession) AddOp(op *protocol.Operation) error {
	if ts.ops == nil {
		return ErrNoSession
	}

	ts.ops = append(ts.ops, op)
	return nil
}

func (ts *TransactionSession) GetOps() ([]*protocol.Operation, error) {
	if ts.ops == nil {
		return nil, ErrNoSession
	}

	return ts.ops, nil
}

func (ts *TransactionSession) IsValid() bool {
	return ts.ops != nil
}
