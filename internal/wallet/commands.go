package wallet

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"

	types "github.com/koinos/koinos-types-golang"
	"github.com/shopspring/decimal"
)

// Hardcoded Koin contract constants
const (
	ReadContractCall      = "chain.read_contract"
	GetAccountNonceCall   = "chain.get_account_nonce"
	SubmitTransactionCall = "chain.submit_transaction"
	KoinSymbol            = "tKOIN"
	KoinPrecision         = 8
	KoinContractID        = "kw96mR+Hh71IWwJoT/2lJXBDl5Q="
	KoinBalanceOfEntry    = types.UInt32(0x15619248)
	KoinTransferEntry     = types.UInt32(0x62efa292)
)

// ----------------------------------------------------------------------------
// Command Declarations
// ----------------------------------------------------------------------------

// All commands should be declared here

// BuildCommands constructs the declarations needed by the parser
func BuildCommands() []*CommandDeclaration {
	var decls []*CommandDeclaration
	decls = append(decls, NewCommandDeclaration("balance", "Check the balance at an address", false, NewBalanceCommand, *NewCommandArg("address", Address)))
	decls = append(decls, NewCommandDeclaration("connect", "Connect to an RPC endpoint", false, NewConnectCommand, *NewCommandArg("url", String)))
	decls = append(decls, NewCommandDeclaration("close", "Close the currently open wallet", false, NewCloseCommand))
	decls = append(decls, NewCommandDeclaration("create", "Create and open a new wallet file", false, NewCreateCommand,
		*NewCommandArg("filename", String), *NewCommandArg("password", String)))
	decls = append(decls, NewCommandDeclaration("disconnect", "Disconnect from RPC endpoint", false, NewDisconnectCommand))
	decls = append(decls, NewCommandDeclaration("generate", "Generate and display a new private key", false, NewGenerateKeyCommand))
	decls = append(decls, NewCommandDeclaration("help", "Show help on a given command", false, NewHelpCommand, *NewCommandArg("command", CmdName)))
	decls = append(decls, NewCommandDeclaration("import", "Import a WIF private key to a new wallet file", false, NewImportCommand, *NewCommandArg("private-key", String),
		*NewCommandArg("filename", String), *NewCommandArg("password", String)))
	decls = append(decls, NewCommandDeclaration("info", "Show the currently opened wallet's address / key", false, NewInfoCommand))
	decls = append(decls, NewCommandDeclaration("open", "Open a wallet file", false, NewOpenCommand,
		*NewCommandArg("filename", String), *NewCommandArg("password", String)))
	decls = append(decls, NewCommandDeclaration("transfer", "Transfer token from an open wallet to a given address", false, NewTransferCommand,
		*NewCommandArg("amount", Amount), *NewCommandArg("address", Address)))
	decls = append(decls, NewCommandDeclaration("exit", "Exit the wallet (quit also works)", false, NewExitCommand))
	decls = append(decls, NewCommandDeclaration("quit", "", true, NewExitCommand))

	return decls
}

// ----------------------------------------------------------------------------
// Command Implementations
// ----------------------------------------------------------------------------

// All commands should be implemented here

// ----------------------------------------------------------------------------
// Balance Command
// ----------------------------------------------------------------------------

// BalanceCommand is a command that checks the balance of an address
type BalanceCommand struct {
	Address *types.AccountType
}

// NewBalanceCommand creates a new balance object
func NewBalanceCommand(inv *CommandParseResult) CLICommand {
	addressString := inv.Args["address"]
	address := types.AccountType(addressString)
	return &BalanceCommand{Address: &address}
}

// Execute fetches the balance
func (c *BalanceCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot check balance", ErrOffline)
	}

	// Setup command execution environment
	contractID, err := ContractStringToID(KoinContractID)
	if err != nil {
		panic("Invalid contract ID")
	}

	balance, err := ee.RPCClient.GetAccountBalance(c.Address, contractID, KoinBalanceOfEntry)

	// Build the result
	dec, err := SatoshiToDecimal(int64(balance), KoinPrecision)
	if err != nil {
		return nil, err
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("%v %s", dec, KoinSymbol))

	return er, nil
}

// ----------------------------------------------------------------------------
// Close Command
// ----------------------------------------------------------------------------

// CloseCommand is a command that closes an open wallet
type CloseCommand struct {
}

// NewCloseCommand creates a new close object
func NewCloseCommand(inv *CommandParseResult) CLICommand {
	return &CloseCommand{}
}

// Execute closes the wallet
func (c *CloseCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot close", ErrWalletClosed)
	}

	// Close the wallet
	ee.Key = nil

	result := NewExecutionResult()
	result.AddMessage("Wallet closed")

	return result, nil
}

// ----------------------------------------------------------------------------
// Connect Command
// ----------------------------------------------------------------------------

// ConnectCommand is a command that connects to an RPC endpoint
type ConnectCommand struct {
	URL string
}

// NewConnectCommand creates a new connect object
func NewConnectCommand(inv *CommandParseResult) CLICommand {
	return &ConnectCommand{URL: inv.Args["url"]}
}

// Execute connects to an RPC endpoint
func (c *ConnectCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	rpc := NewKoinosRPCClient(c.URL)
	ee.RPCClient = rpc

	// TODO: Ensure connection (some sort of ping?)
	// Issue #20

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Connected to endpoint %s", c.URL))

	return result, nil
}

// ----------------------------------------------------------------------------
// Disonnect Command
// ----------------------------------------------------------------------------

// DisconnectCommand is a command that disconnects from an RPC endpoint
type DisconnectCommand struct {
}

// NewDisconnectCommand creates a new disconnect object
func NewDisconnectCommand(inv *CommandParseResult) CLICommand {
	return &DisconnectCommand{}
}

// Execute disconnects from an RPC endpoint
func (c *DisconnectCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot disconnect", ErrOffline)
	}

	// Disconnect from the RPC endpoint
	ee.RPCClient = nil

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Disconnected"))

	return result, nil
}

// ----------------------------------------------------------------------------
// Exit Command
// ----------------------------------------------------------------------------

// ExitCommand is a command that exits the wallet
type ExitCommand struct {
}

// NewExitCommand creates a new exit object
func NewExitCommand(inv *CommandParseResult) CLICommand {
	return &ExitCommand{}
}

// Execute exits the wallet
func (c *ExitCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	os.Exit(0)
	return nil, nil
}

// ----------------------------------------------------------------------------
// Generate Key Command
// ----------------------------------------------------------------------------

// GenerateKeyCommand is a command that exits the wallet
type GenerateKeyCommand struct {
}

// NewGenerateKeyCommand creates a new exit object
func NewGenerateKeyCommand(inv *CommandParseResult) CLICommand {
	return &GenerateKeyCommand{}
}

// Execute exits the wallet
func (c *GenerateKeyCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	k, err := GenerateKoinosKey()
	if err != nil {
		return nil, err
	}

	result := NewExecutionResult()
	result.AddMessage("New key generated. This is only shown once, make sure to record this information.")
	result.AddMessage(fmt.Sprintf("Address: %s", k.Address()))
	result.AddMessage(fmt.Sprintf("Private: %s", k.Private()))

	return result, nil
}

// ----------------------------------------------------------------------------
// Create
// ----------------------------------------------------------------------------

// CreateCommand is a command that creates a new wallet
type CreateCommand struct {
	Filename string
	Password string
}

// NewCreateCommand creates a new create object
func NewCreateCommand(inv *CommandParseResult) CLICommand {
	return &CreateCommand{Filename: inv.Args["filename"], Password: inv.Args["password"]}
}

// Execute creates a new wallet
func (c *CreateCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {

	// Check if the wallet already exists
	if _, err := os.Stat(c.Filename); !os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrWalletExists, c.Filename)
	}

	// Generate new key
	key, err := GenerateKoinosKey()
	if err != nil {
		return nil, err
	}

	// Create the wallet file
	file, err := os.Create(c.Filename)
	if err != nil {
		return nil, err
	}

	// Write the key to the wallet file
	err = CreateWalletFile(file, c.Password, key.PrivateBytes())
	if err != nil {
		return nil, err
	}

	// Set the wallet keys
	ee.Key = key

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Created and opened new wallet: %s", c.Filename))
	result.AddMessage(fmt.Sprintf("Address: %s", key.Address()))
	result.AddMessage("Use the info command to see more information")

	return result, nil
}

// ----------------------------------------------------------------------------
// Import
// ----------------------------------------------------------------------------

// ImportCommand is a command that imports a private key to a wallet
type ImportCommand struct {
	Filename   string
	Password   string
	PrivateKey string
}

// NewImportCommand creates a new import object
func NewImportCommand(inv *CommandParseResult) CLICommand {
	return &ImportCommand{Filename: inv.Args["filename"], Password: inv.Args["password"], PrivateKey: inv.Args["private-key"]}
}

// Execute creates a new wallet
func (c *ImportCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	// Check if the wallet already exists
	if _, err := os.Stat(c.Filename); !os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrWalletExists, c.Filename)
	}

	// Convert the private key to bytes
	keyBytes, err := DecodeWIF(c.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Create the key
	key, err := NewKoinosKeysFromBytes(keyBytes)
	if err != nil {
		return nil, err
	}

	// Create the wallet file
	file, err := os.Create(c.Filename)
	if err != nil {
		return nil, err
	}

	// Write the key to the wallet file
	err = CreateWalletFile(file, c.Password, key.PrivateBytes())
	if err != nil {
		return nil, err
	}

	// Set the wallet keys
	ee.Key = key

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Created and opened new wallet: %s", c.Filename))
	result.AddMessage(fmt.Sprintf("Address: %s", key.Address()))
	result.AddMessage("Use the info command to see more information")

	return result, nil
}

// ----------------------------------------------------------------------------
// Info Command
// ----------------------------------------------------------------------------

// InfoCommand is a command that shows the currently opened wallet's address and private key
type InfoCommand struct {
}

// NewInfoCommand creates a new info command object
func NewInfoCommand(inv *CommandParseResult) CLICommand {
	return &InfoCommand{}
}

// Execute shows wallet info
func (c *InfoCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot show info", ErrWalletClosed)
	}

	result := NewExecutionResult()
	result.AddMessage("Wallet information:")
	result.AddMessage(fmt.Sprintf("Address: %s", ee.Key.Address()))
	result.AddMessage(fmt.Sprintf("Private: %s", ee.Key.Private()))

	return result, nil
}

// ----------------------------------------------------------------------------
// Help
// ----------------------------------------------------------------------------

// OpenCommand is a command that opens a wallet file
type HelpCommand struct {
	Command string
}

// NewOpenCommand creates a new open command object
func NewHelpCommand(inv *CommandParseResult) CLICommand {
	return &HelpCommand{Command: inv.Args["command"]}
}

// Execute opens a wallet
func (c *HelpCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	decl, ok := ee.Parser.Name2Command[string(c.Command)]

	if !ok {
		return nil, fmt.Errorf("%w: cannot show help for %s", ErrUnknownCommand, c.Command)
	}

	result := NewExecutionResult()
	result.AddMessage(decl.Description)
	result.AddMessage("Usage:")
	result.AddMessage(decl.String())

	return result, nil
}

// ----------------------------------------------------------------------------
// Open
// ----------------------------------------------------------------------------

// OpenCommand is a command that opens a wallet file
type OpenCommand struct {
	Filename string
	Password string
}

// NewOpenCommand creates a new open command object
func NewOpenCommand(inv *CommandParseResult) CLICommand {
	return &OpenCommand{Filename: inv.Args["filename"], Password: inv.Args["password"]}
}

// Execute opens a wallet
func (c *OpenCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	// Open the wallet file
	file, err := os.Open(c.Filename)
	if err != nil {
		return nil, err
	}

	// Read the wallet file
	keyBytes, err := ReadWalletFile(file, c.Password)
	if err != nil {
		return nil, fmt.Errorf("%w: check your password", ErrWalletDecrypt)
	}

	// Create the key object
	key, err := NewKoinosKeysFromBytes(keyBytes)
	if err != nil {
		return nil, err
	}

	// Set the wallet keys
	ee.Key = key

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Opened wallet: %s", c.Filename))

	return result, nil
}

// ----------------------------------------------------------------------------
// Transfer
// ----------------------------------------------------------------------------

// TransferCommand is a command that closes an open wallet
type TransferCommand struct {
	Address *types.AccountType
	Amount  string
}

// NewTransferCommand creates a new close object
func NewTransferCommand(inv *CommandParseResult) CLICommand {
	addressString := inv.Args["address"]
	address := types.AccountType(addressString)
	return &TransferCommand{Address: &address, Amount: inv.Args["amount"]}
}

// Execute transfers token
func (c *TransferCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot transfer", ErrWalletClosed)
	}

	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot transfer", ErrOffline)
	}

	// Convert the amount to a decimal
	dAmount, err := decimal.NewFromString(c.Amount)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidAmount, err.Error())
	}

	// Convert the amount to satoshis
	sAmount, err := DecimalToSatoshi(&dAmount, KoinPrecision)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidAmount, err.Error())
	}

	// Ensure a transfer greater than zero
	if sAmount <= 0 {
		return nil, fmt.Errorf("%w: cannot transfer %d %s", ErrInvalidAmount, sAmount, KoinSymbol)
	}

	// Fetch the account's nonce
	myAddress := types.AccountType(ee.Key.Address())
	nonce, err := ee.RPCClient.GetAccountNonce(&myAddress)
	if err != nil {
		return nil, err
	}

	// Setup command execution environment
	contractID, err := ContractStringToID(KoinContractID)
	if err != nil {
		panic("Invalid contract ID")
	}

	// Create the operation
	callContractOp := types.NewCallContractOperation()
	callContractOp.ContractID = *contractID
	callContractOp.EntryPoint = KoinTransferEntry

	// Serialize and assign the args
	vb := types.NewVariableBlob()
	vb = myAddress.Serialize(vb)
	vb = c.Address.Serialize(vb)
	tAmount := types.UInt64(sAmount)
	vb = tAmount.Serialize(vb)
	callContractOp.Args = *vb

	// Create a variant operation and assign the call contract operation
	op := types.NewOperation()
	op.Value = callContractOp

	// Create the transaction
	transaction := types.NewTransaction()
	transaction.ActiveData.Native.Operations = append(transaction.ActiveData.Native.Operations, *op)
	transaction.ActiveData.Native.Nonce = nonce
	rLimit, err := types.NewUInt128FromString("1000000")
	if err != nil {
		return nil, err
	}
	transaction.ActiveData.Native.ResourceLimit = *rLimit

	// Calculate the transaction ID
	activeDataBytes := transaction.ActiveData.Serialize(types.NewVariableBlob())
	sha256Hasher := sha256.New()
	sha256Hasher.Write(*activeDataBytes)
	transactionID := sha256Hasher.Sum(nil)
	transaction.ID.ID = 0x12 // SHA2_256_ID
	transaction.ID.Digest = transactionID

	// Sign the transaction
	err = SignTransaction(ee.Key.PrivateBytes(), transaction)

	if err != nil {
		return nil, err
	}

	// Submit the transaction
	params := types.NewSubmitTransactionRequest()
	params.Transaction = *transaction

	// Make the rpc call
	var cResp types.SubmitTransactionResponse
	err = ee.RPCClient.Call(SubmitTransactionCall, params, &cResp)
	if err != nil {
		return nil, err
	}

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Transferring %s %s to %s", dAmount, KoinSymbol, *c.Address))

	return result, nil
}
