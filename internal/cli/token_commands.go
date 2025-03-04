package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/btcsuite/btcutil/base58"
	"github.com/koinos/koinos-cli/internal/cliutil"
	"github.com/koinos/koinos-proto-golang/v2/koinos/protocol"
	"github.com/koinos/koinos-proto-golang/v2/koinos/standards/kcs4"
	util "github.com/koinos/koinos-util-golang/v2"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"
)

const (
	TokenBalanceOfEntry   = uint32(0x5c721497)
	TokenTransferEntry    = uint32(0x27f576ca)
	TokenTotalSupplyEntry = uint32(0xb0da3934)
	TokenSymbolEntry      = uint32(0xb76a7ca1)
	TokenDecimalsEntry    = uint32(0xee80fd2f)
	TokenAllowanceEntry   = uint32(0x32f09fa1)
	TokenAllowancesEntry  = uint32(0x8fa16456)
	TokenApproveEntry     = uint32(0x74e21680)
)

func retrieveSymbol(ctx context.Context, client *cliutil.KoinosRPCClient, contractID []byte) (*string, error) {
	symbolArguments := kcs4.SymbolArguments{}

	args, err := proto.Marshal(&symbolArguments)
	if err != nil {
		return nil, err
	}

	resp, err := client.ReadContract(ctx, args, contractID, TokenSymbolEntry)
	if err != nil {
		return nil, err
	}

	symbolResult := &kcs4.SymbolResult{}
	err = proto.Unmarshal(resp.GetResult(), symbolResult)
	if err != nil {
		return nil, err
	}

	return &symbolResult.Value, nil
}

func retrieveDecimals(ctx context.Context, client *cliutil.KoinosRPCClient, contractID []byte) (*int, error) {
	decimalsArguments := kcs4.DecimalsArguments{}

	args, err := proto.Marshal(&decimalsArguments)
	if err != nil {
		return nil, err
	}

	resp, err := client.ReadContract(ctx, args, contractID, TokenDecimalsEntry)
	if err != nil {
		return nil, err
	}

	decimalsResult := &kcs4.DecimalsResult{}
	err = proto.Unmarshal(resp.GetResult(), decimalsResult)
	if err != nil {
		return nil, err
	}

	value := int(decimalsResult.Value)

	return &value, nil
}

func retrieveBalance(ctx context.Context, client *cliutil.KoinosRPCClient, contractID []byte, address []byte) (*uint64, error) {
	balanceOfArguments := kcs4.BalanceOfArguments{}
	balanceOfArguments.Owner = address

	args, err := proto.Marshal(&balanceOfArguments)
	if err != nil {
		return nil, err
	}

	resp, err := client.ReadContract(ctx, args, contractID, TokenBalanceOfEntry)
	if err != nil {
		return nil, err
	}

	balanceOfResult := &kcs4.BalanceOfResult{}
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
	cmd = NewCommandDeclaration(fmt.Sprintf("%s.transfer", c.Name), "Transfers the token", false, NewTransferCommand, *NewCommandArg("to", AddressArg), *NewCommandArg("amount", AmountArg), *NewOptionalCommandArg("memo", StringArg))
	ee.Parser.Commands.AddCommand(cmd)

	NewAllowanceCommand := func(inv *CommandParseResult) Command {
		return NewTokenAllowanceCommand(inv, contractID, *precision, *symbol)
	}
	cmd = NewCommandDeclaration(fmt.Sprintf("%s.allowance", c.Name), "Returns a token allowance", false, NewAllowanceCommand, *NewCommandArg("spender", AddressArg), *NewOptionalCommandArg("owner", AddressArg))
	ee.Parser.Commands.AddCommand(cmd)

	NewAllowancesCommand := func(inv *CommandParseResult) Command {
		return NewTokenAllowancesCommand(inv, contractID, *precision, *symbol)
	}
	cmd = NewCommandDeclaration(fmt.Sprintf("%s.allowances", c.Name), "Returns token allowances", false, NewAllowancesCommand, *NewOptionalCommandArg("start", AddressArg), *NewOptionalCommandArg("limit", UIntArg), *NewOptionalCommandArg("owner", AddressArg))
	ee.Parser.Commands.AddCommand(cmd)

	NewApproveCommand := func(inv *CommandParseResult) Command {
		return NewTokenApproveCommand(inv, contractID, *precision, *symbol)
	}
	cmd = NewCommandDeclaration(fmt.Sprintf("%s.approve", c.Name), "Approves an address to spend token", false, NewApproveCommand, *NewCommandArg("spender", AddressArg), *NewCommandArg("amount", AmountArg), *NewOptionalCommandArg("memo", StringArg))
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

	totalSupplyArguments := kcs4.TotalSupplyArguments{}

	args, err := proto.Marshal(&totalSupplyArguments)
	if err != nil {
		return nil, err
	}

	resp, err := ee.RPCClient.ReadContract(ctx, args, c.ContractID, TokenTotalSupplyEntry)
	if err != nil {
		return nil, err
	}

	totalSupplyResult := &kcs4.TotalSupplyResult{}
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
	Memo       *string
	ContractID []byte
	Precision  int
	Symbol     string
}

// NewTokenTransferCommand instantiates the command to transfer tokens
func NewTokenTransferCommand(inv *CommandParseResult, contractID []byte, precision int, symbol string) Command {
	return &TokenTransferCommand{Address: *inv.Args["to"], Amount: *inv.Args["amount"], Memo: inv.Args["memo"], ContractID: contractID, Precision: precision, Symbol: symbol}
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

	transferArgs := &kcs4.TransferArguments{
		From:  walletAddress,
		To:    toAddress,
		Value: uint64(satoshiAmount),
	}

	if c.Memo != nil {
		transferArgs.Memo = c.Memo
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

// TokenAllowanceCommand is a command that returns a token allowance
type TokenAllowanceCommand struct {
	Spender    string
	Owner      *string
	ContractID []byte
	Precision  int
	Symbol     string
}

// NewTokenAllowanceCommand instantiates the command to return an allowance
func NewTokenAllowanceCommand(inv *CommandParseResult, contractID []byte, precision int, symbol string) Command {
	return &TokenAllowanceCommand{Spender: *inv.Args["spender"], ContractID: contractID, Precision: precision, Symbol: symbol}
}

// Execute the token allowance
func (c *TokenAllowanceCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot check allowance", cliutil.ErrOffline)
	}

	var owner []byte
	if c.Owner == nil {
		if !ee.IsWalletOpen() {
			return nil, fmt.Errorf("%w: must give an owner address", cliutil.ErrWalletClosed)
		}

		owner = ee.Key.AddressBytes()
	} else {
		owner = base58.Decode(*c.Owner)
		if len(owner) == 0 {
			return nil, errors.New("could not parse owner address")
		}
	}

	spender := base58.Decode(c.Spender)
	if len(spender) == 0 {
		return nil, errors.New("could not parse spender address")
	}

	allowanceArguments := kcs4.AllowanceArguments{
		Owner:   owner,
		Spender: spender,
	}

	args, err := proto.Marshal(&allowanceArguments)
	if err != nil {
		return nil, err
	}

	resp, err := ee.RPCClient.ReadContract(ctx, args, c.ContractID, TokenAllowanceEntry)
	if err != nil {
		return nil, err
	}

	allowanceResult := kcs4.AllowanceResult{}
	err = proto.Unmarshal(resp.GetResult(), &allowanceResult)
	if err != nil {
		return nil, err
	}

	dec, err := util.SatoshiToDecimal(allowanceResult.Value, c.Precision)
	if err != nil {
		return nil, err
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("%s %s", dec, c.Symbol))

	return er, nil
}

// AllowancesCommand is a command that returns token allowances
type TokenAllowancesCommand struct {
	Start      *string
	Limit      *string
	Owner      *string
	ContractID []byte
	Precision  int
	Symbol     string
}

// NewAllowanceCommand instantiates the command to return an allowance
func NewTokenAllowancesCommand(inv *CommandParseResult, contractID []byte, precision int, symbol string) Command {
	return &TokenAllowancesCommand{Start: inv.Args["start"], Limit: inv.Args["limit"], Owner: inv.Args["owner"], ContractID: contractID, Precision: precision, Symbol: symbol}
}

// Execute the token allowance
func (c *TokenAllowancesCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot check allowance", cliutil.ErrOffline)
	}

	var owner []byte
	if c.Owner == nil {
		if !ee.IsWalletOpen() {
			return nil, fmt.Errorf("%w: must give an owner address", cliutil.ErrWalletClosed)
		}

		owner = ee.Key.AddressBytes()
	} else {
		owner = base58.Decode(*c.Owner)
		if len(owner) == 0 {
			return nil, errors.New("could not parse owner address")
		}
	}

	limit := int32(10)
	if c.Limit != nil {
		limit64, err := strconv.ParseUint(*c.Limit, 10, 31)
		if err != nil {
			return nil, err
		}

		limit = int32(limit64)
	}

	getAllowancesArgs := kcs4.GetAllowancesArguments{
		Owner: owner,
		Limit: limit,
	}

	if c.Start != nil && len(*c.Start) > 0 {
		start := base58.Decode(*c.Start)
		if len(start) == 0 {
			return nil, errors.New("could not parse start address")
		}

		getAllowancesArgs.Start = start
	}

	args, err := proto.Marshal(&getAllowancesArgs)
	if err != nil {
		return nil, err
	}

	resp, err := ee.RPCClient.ReadContract(ctx, args, c.ContractID, TokenAllowancesEntry)
	if err != nil {
		return nil, err
	}

	getAllowancesResult := kcs4.GetAllowancesResult{}
	err = proto.Unmarshal(resp.GetResult(), &getAllowancesResult)
	if err != nil {
		return nil, err
	}

	er := NewExecutionResult()
	er.AddMessage("Allowances:")

	for _, allowance := range getAllowancesResult.Allowances {
		dec, err := util.SatoshiToDecimal(allowance.Value, c.Precision)
		if err != nil {
			return nil, err
		}

		er.AddMessage(fmt.Sprintf(" - %34s: %s %s", base58.Encode(allowance.Spender), dec, c.Symbol))
	}

	return er, nil
}

// TokenAllowanceCommand is a command that returns a token allowance
type TokenApproveCommand struct {
	Spender    string
	Amount     string
	Memo       *string
	ContractID []byte
	Precision  int
	Symbol     string
}

// NewTokenAllowanceCommand instantiates the command to return an allowance
func NewTokenApproveCommand(inv *CommandParseResult, contractID []byte, precision int, symbol string) Command {
	return &TokenApproveCommand{Spender: *inv.Args["spender"], Amount: *inv.Args["amount"], Memo: inv.Args["memo"], ContractID: contractID, Precision: precision, Symbol: symbol}
}

// Execute the token allowance
func (c *TokenApproveCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
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

	walletAddress := ee.Key.AddressBytes()

	spender := base58.Decode(c.Spender)
	if len(spender) == 0 {
		return nil, errors.New("could not parse spender address")
	}

	approveArgs := &kcs4.ApproveArguments{
		Owner:   walletAddress,
		Spender: spender,
		Value:   uint64(satoshiAmount),
	}

	if c.Memo != nil {
		approveArgs.Memo = c.Memo
	}

	args, err := proto.Marshal(approveArgs)
	if err != nil {
		return nil, err
	}

	op := &protocol.Operation{
		Op: &protocol.Operation_CallContract{
			CallContract: &protocol.CallContractOperation{
				ContractId: c.ContractID,
				EntryPoint: TokenApproveEntry,
				Args:       args,
			},
		},
	}

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Approving %s for %s %s", c.Spender, decimalAmount, c.Symbol))

	err = ee.Session.AddOperation(op, fmt.Sprintf("Approve %s for %s %s", c.Spender, decimalAmount, c.Symbol))
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
