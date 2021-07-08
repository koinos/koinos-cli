package internal

import (
	"context"
	"fmt"

	types "github.com/koinos/koinos-types-golang"
	"github.com/shopspring/decimal"
	"github.com/ybbus/jsonrpc/v2"
)

const (
	ReadContractCall = "chain.read_contract"
	KoinSymbol       = "tKOIN"
)

type CLICommand interface {
	Execute(ctx context.Context, ee ExecutionEnvironment) (ExecutionResult, error)
}

type ExecutionResult struct {
	Message string
}

// ExecutionEnvironment is a struct that holds the environment for command execution.
type ExecutionEnvironment struct {
	RPCClient          jsonrpc.RPCClient
	KoinContractID     *types.ContractIDType
	KoinBalanceOfEntry types.UInt32
}

// ----------------------------------------------------------------------------
// Balance
// ----------------------------------------------------------------------------

type BalanceCommand struct {
	Address *types.AccountType
}

func (c *BalanceCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	// Build the contract request
	params := types.NewReadContractRequest()
	params.ContractID = *ee.KoinContractID
	params.EntryPoint = ee.KoinBalanceOfEntry
	// Serialize the args
	vb := types.NewVariableBlob()
	vb = c.Address.Serialize(vb)
	params.Args = *vb

	// Make the rpc call
	resp, err := ee.RPCClient.Call(ReadContractCall, params)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	// Fetch the contract response
	var c_resp types.ReadContractResponse
	err = resp.GetObject(&c_resp)
	if err != nil {
		return nil, err
	}

	_, balance, err := types.DeserializeUInt64(&c_resp.Result)
	if err != nil {
		return nil, err
	}

	// Build the result
	er := ExecutionResult{Message: fmt.Sprintf("%v %s", koinToDecimal(balance), KoinSymbol)}

	return &er, nil
}

func koinToDecimal(balance *types.UInt64) *decimal.Decimal {
	v := decimal.NewFromInt(int64(*balance)).Div(decimal.NewFromInt(100000000))
	return &v
}
