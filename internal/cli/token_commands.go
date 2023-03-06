package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/btcsuite/btcutil/base58"
	"github.com/koinos/koinos-cli/internal/cliutil"
	"github.com/koinos/koinos-proto-golang/koinos/contracts/token"
	"github.com/koinos/koinos-proto-golang/koinos/protocol"
	util "github.com/koinos/koinos-util-golang"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
)

const (
	TokenBalanceOfEntry   = uint32(0x5c721497)
	TokenTransferEntry    = uint32(0x27f576ca)
	TokenTotalSupplyEntry = uint32(0xb0da3934)
	TokenSymbolEntry      = uint32(0xb76a7ca1)
	TokenDecimalsEntry    = uint32(0xee80fd2f)
)

func retrieveSymbol(ctx context.Context, client *cliutil.KoinosRPCClient, contractID []byte) (*string, error) {
	symbolArguments := token.SymbolArguments{}

	args, err := proto.Marshal(&symbolArguments)
	if err != nil {
		return nil, err
	}

	resp, err := client.ReadContract(ctx, args, contractID, TokenSymbolEntry)
	if err != nil {
		return nil, err
	}

	symbolResult := &token.SymbolResult{}
	err = proto.Unmarshal(resp.GetResult(), symbolResult)
	if err != nil {
		return nil, err
	}

	return &symbolResult.Value, nil
}

func retrieveDecimals(ctx context.Context, client *cliutil.KoinosRPCClient, contractID []byte) (*int, error) {
	decimalsArguments := token.DecimalsArguments{}

	args, err := proto.Marshal(&decimalsArguments)
	if err != nil {
		return nil, err
	}

	resp, err := client.ReadContract(ctx, args, contractID, TokenDecimalsEntry)
	if err != nil {
		return nil, err
	}

	decimalsResult := &token.DecimalsResult{}
	err = proto.Unmarshal(resp.GetResult(), decimalsResult)
	if err != nil {
		return nil, err
	}

	value := int(decimalsResult.Value)

	return &value, nil
}

func retrieveBalance(ctx context.Context, client *cliutil.KoinosRPCClient, contractID []byte, address []byte) (*uint64, error) {
	balanceOfArguments := token.BalanceOfArguments{}
	balanceOfArguments.Owner = address

	args, err := proto.Marshal(&balanceOfArguments)
	if err != nil {
		return nil, err
	}

	resp, err := client.ReadContract(ctx, args, contractID, TokenBalanceOfEntry)
	if err != nil {
		return nil, err
	}

	balanceOfResult := &token.BalanceOfResult{}
	err = proto.Unmarshal(resp.GetResult(), balanceOfResult)
	if err != nil {
		return nil, err
	}

	return &balanceOfResult.Value, nil
}

// ----------------------------------------------------------------------------
// RegisterToken
// ----------------------------------------------------------------------------

// RegisterTokenCommand is a command that registers token commands
type RegisterTokenCommand struct {
	Name      string
	Address   string
	Symbol    *string
	Precision *string
}

// NewRegisterTokenCommand instantiates the command to register tokens
func NewRegisterTokenCommand(inv *CommandParseResult) Command {
	return &RegisterTokenCommand{Name: *inv.Args["name"], Address: *inv.Args["address"], Symbol: inv.Args["symbol"], Precision: inv.Args["precision"]}
}

// Execute registers token commands
func (c *RegisterTokenCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if (c.Symbol == nil || c.Precision == nil) && !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot retrieve symbol and precision", cliutil.ErrOffline)
	}

	if ee.Contracts.Contains(c.Name) {
		return nil, fmt.Errorf("%w: token %s already exists", cliutil.ErrContract, c.Name)
	}

	_, err := ee.Parser.parseCommandName([]byte(c.Name))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid characters in token name %s", cliutil.ErrContract, err)
	}

	contractID := base58.Decode(c.Address)
	if len(contractID) == 0 {
		return nil, errors.New("could not parse contract ID")
	}

	var symbol *string
	if c.Symbol == nil {
		symbol, err = retrieveSymbol(ctx, ee.RPCClient, contractID)
		if err != nil {
			return nil, err
		}
	} else {
		symbol = c.Symbol
	}

	var precision *int
	if c.Precision == nil {
		precision, err = retrieveDecimals(ctx, ee.RPCClient, contractID)
		if err != nil {
			return nil, err
		}
	} else {
		precision = new(int)
		*precision, err = strconv.Atoi(*c.Precision)
		if err != nil {
			return nil, err
		}
	}

	NewBalanceOfCommand := func(inv *CommandParseResult) Command {
		return NewTokenBalanceCommand(inv, contractID, *precision, *symbol)
	}
	cmd := NewCommandDeclaration(fmt.Sprintf("%s.balance_of", c.Name), "Checks the balance at an address", false, NewBalanceOfCommand, *NewOptionalCommandArg("address", AddressArg))
	ee.Parser.Commands.AddCommand(cmd)

	NewTotalSupplyCommand := func(inv *CommandParseResult) Command {
		return NewTokenTotalSupplyCommand(inv, contractID, *precision, *symbol)
	}
	cmd = NewCommandDeclaration(fmt.Sprintf("%s.total_supply", c.Name), "Checks the token total supply", false, NewTotalSupplyCommand)
	ee.Parser.Commands.AddCommand(cmd)

	NewTransferCommand := func(inv *CommandParseResult) Command {
		return NewTokenTransferCommand(inv, contractID, *precision, *symbol)
	}
	cmd = NewCommandDeclaration(fmt.Sprintf("%s.transfer", c.Name), "Transfers the token", false, NewTransferCommand, *NewCommandArg("to", AddressArg), *NewCommandArg("amount", AmountArg))
	ee.Parser.Commands.AddCommand(cmd)

	err = ee.Contracts.Add(c.Name, c.Address, nil, nil)
	if err != nil {
		return nil, err
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("Token '%s' at address %s registered", c.Name, c.Address))
	return er, nil
}

// ----------------------------------------------------------------------------
// TokenBalance
// ----------------------------------------------------------------------------

// TokenBalanceCommand is a command that retrieves the balance of a particular token
type TokenBalanceCommand struct {
	Address    *string
	ContractID []byte
	Precision  int
	Symbol     string
}

// NewTokenBalanceCommand instantiates the command to retrieve a token balance
func NewTokenBalanceCommand(inv *CommandParseResult, contractID []byte, precision int, symbol string) Command {
	return &TokenBalanceCommand{Address: inv.Args["address"], ContractID: contractID, Precision: precision, Symbol: symbol}
}

// Execute retrieves token balance
func (c *TokenBalanceCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot check balance", cliutil.ErrOffline)
	}

	var address []byte
	if c.Address == nil {
		if !ee.IsWalletOpen() {
			return nil, fmt.Errorf("%w: must give an address", cliutil.ErrWalletClosed)
		}

		address = ee.Key.AddressBytes()
	} else {
		address = base58.Decode(*c.Address)
		if len(address) == 0 {
			return nil, errors.New("could not parse address")
		}
	}

	balance, err := retrieveBalance(ctx, ee.RPCClient, c.ContractID, address)
	if err != nil {
		return nil, err
	}

	dec, err := util.SatoshiToDecimal(*balance, c.Precision)
	if err != nil {
		return nil, err
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("%v %s", dec, c.Symbol))

	return er, nil
}

// ----------------------------------------------------------------------------
// TokenTotalSupply
// ----------------------------------------------------------------------------

// TokenTotalSupplyCommand is a command that retrieves the total supply of a particular token
type TokenTotalSupplyCommand struct {
	ContractID []byte
	Precision  int
	Symbol     string
}

// NewTokenTotalSupplyCommand instantiates the command to retrieve the total supply of a token
func NewTokenTotalSupplyCommand(inv *CommandParseResult, contractID []byte, precision int, symbol string) Command {
	return &TokenTotalSupplyCommand{ContractID: contractID, Precision: precision, Symbol: symbol}
}

// Execute retrieves the token total supply
func (c *TokenTotalSupplyCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot check total supply", cliutil.ErrOffline)
	}

	totalSupplyArguments := token.TotalSupplyArguments{}

	args, err := proto.Marshal(&totalSupplyArguments)
	if err != nil {
		return nil, err
	}

	resp, err := ee.RPCClient.ReadContract(ctx, args, c.ContractID, TokenTotalSupplyEntry)
	if err != nil {
		return nil, err
	}

	totalSupplyResult := &token.TotalSupplyResult{}
	err = proto.Unmarshal(resp.GetResult(), totalSupplyResult)
	if err != nil {
		return nil, err
	}

	dec, err := util.SatoshiToDecimal(totalSupplyResult.GetValue(), c.Precision)
	if err != nil {
		return nil, err
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("%v %s", dec, c.Symbol))

	return er, nil
}

// ----------------------------------------------------------------------------
// TokenTransfer
// ----------------------------------------------------------------------------

// TokenTransferCommand is a command that transfers tokens
type TokenTransferCommand struct {
	Address    string
	Amount     string
	ContractID []byte
	Precision  int
	Symbol     string
}

// NewTokenTransferCommand instantiates the command to transfer tokens
func NewTokenTransferCommand(inv *CommandParseResult, contractID []byte, precision int, symbol string) Command {
	return &TokenTransferCommand{Address: *inv.Args["to"], Amount: *inv.Args["amount"], ContractID: contractID, Precision: precision, Symbol: symbol}
}

// Execute the token transfer
func (c *TokenTransferCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot transfer", cliutil.ErrWalletClosed)
	}

	if !ee.IsOnline() && !ee.Session.IsValid() {
		return nil, fmt.Errorf("%w: cannot transfer", cliutil.ErrOffline)
	}

	decimalAmount, err := decimal.NewFromString(c.Amount)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidAmount, err.Error())
	}

	satoshiAmount, err := util.DecimalToSatoshi(&decimalAmount, c.Precision)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidAmount, err.Error())
	}

	if satoshiAmount <= 0 {
		minimalAmount, _ := util.SatoshiToDecimal(1, c.Precision)
		return nil, fmt.Errorf("%w: cannot transfer %s %s, amount should be greater than minimal %s (1e-%d) %s", cliutil.ErrInvalidAmount, decimalAmount, c.Symbol, minimalAmount, c.Precision, c.Symbol)
	}

	walletAddress := ee.Key.AddressBytes()

	if ee.IsOnline() {
		balance, err := retrieveBalance(ctx, ee.RPCClient, c.ContractID, walletAddress)
		if err != nil {
			return nil, err
		}

		decimalBalance, err := util.SatoshiToDecimal(*balance, c.Precision)
		if err != nil {
			return nil, err
		}

		if *balance < satoshiAmount {
			return nil, fmt.Errorf("%w: insufficient balance %s %s on opened wallet %s, cannot transfer %s %s", cliutil.ErrInvalidAmount, decimalBalance, c.Symbol, base58.Encode(walletAddress), decimalAmount, c.Symbol)
		}
	}

	toAddress := base58.Decode(c.Address)
	if len(toAddress) == 0 {
		return nil, errors.New("could not parse address")
	}

	transferArgs := &token.TransferArguments{
		From:  walletAddress,
		To:    toAddress,
		Value: uint64(satoshiAmount),
	}

	args, err := proto.Marshal(transferArgs)
	if err != nil {
		return nil, err
	}

	op := &protocol.Operation{
		Op: &protocol.Operation_CallContract{
			CallContract: &protocol.CallContractOperation{
				ContractId: c.ContractID,
				EntryPoint: TokenTransferEntry,
				Args:       args,
			},
		},
	}

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Transferring %s %s to %s", decimalAmount, c.Symbol, c.Address))

	err = ee.Session.AddOperation(op, fmt.Sprintf("Transfer %s %s to %s", decimalAmount, c.Symbol, c.Address))
	if err == nil {
		result.AddMessage("Adding operation to transaction session")
	}
	if err != nil {
		err := ee.SubmitTransaction(ctx, result, op)
		if err != nil {
			return result, fmt.Errorf("cannot transfer, %w", err)
		}
	}

	return result, nil
}
