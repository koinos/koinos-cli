package wallet

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"

	"github.com/btcsuite/btcutil/base58"
	"github.com/koinos/koinos-cli-wallet/internal/kjsonrpc"
	"github.com/koinos/koinos-proto-golang/koinos/canonical"
	"github.com/koinos/koinos-proto-golang/koinos/contracts/token"
	"github.com/koinos/koinos-proto-golang/koinos/protocol"
	"github.com/koinos/koinos-proto-golang/koinos/rpc/chain"
	"github.com/multiformats/go-multihash"
	"github.com/shopspring/decimal"

	"github.com/koinos/koinos-cli-wallet/internal/util"
)

// Hardcoded Koin contract constants
const (
	KoinSymbol         = "tKOIN"
	ManaSymbol         = "mana"
	KoinPrecision      = 8
	KoinContractID     = "5MvF7o3GhJsoRfTZZbeGSe5WzVuVYtBV"
	KoinBalanceOfEntry = uint32(0x15619248)
	KoinTransferEntry  = uint32(0x62efa292)
)

// Hardcoded Multihash constants.
const (
	RIPEMD128 = 0x1052
	RIPEMD160 = 0x1053
	RIPEMD256 = 0x1054
	RIPEMD320 = 0x1055
)

// CommandSet represents a set of commands for the parser
type CommandSet struct {
	Commands     []*CommandDeclaration
	Name2Command map[string]*CommandDeclaration

	// Revision is incremented every time a command is added or removed, for simple versioning
	Revision int
}

// NewCommandSet creates a new command set
func NewCommandSet() *CommandSet {
	cs := &CommandSet{}
	cs.Commands = make([]*CommandDeclaration, 0)
	cs.Name2Command = make(map[string]*CommandDeclaration)

	return cs
}

// AddCommand add a command to the command set
func (cs *CommandSet) AddCommand(decl *CommandDeclaration) {
	cs.Commands = append(cs.Commands, decl)
	cs.Name2Command[decl.Name] = decl
	cs.Revision++
}

// List returns an alphabetized list of commands. The pretty argument makes it return the commands in neat columns with the descriptions
func (cs *CommandSet) List(pretty bool) []string {
	names := make([]string, 0)
	longest := 0

	// Compile the names, and find the longest
	for _, c := range cs.Commands {
		if c.Hidden {
			continue
		}

		names = append(names, c.Name)
		if len(c.Name) > longest {
			longest = len(c.Name)
		}
	}

	// Alphabetize the list
	sort.Strings(names)

	// If no pretty output, just return the list
	if !pretty {
		return names
	}

	// If pretty output, add descriptions
	o := make([]string, 0)
	for _, name := range names {
		o = append(o, fmt.Sprintf("%*s - %s", -longest, name, cs.Name2Command[name].Description))
	}

	return o
}

// ----------------------------------------------------------------------------
// Command Declarations
// ----------------------------------------------------------------------------

// All commands should be declared here

// NewKoinosCommandSet creates the base set of commands used by the wallet
func NewKoinosCommandSet() *CommandSet {
	cs := NewCommandSet()

	cs.AddCommand(NewCommandDeclaration("address", "Show the currently opened wallet's address", false, NewAddressCommand))
	cs.AddCommand(NewCommandDeclaration("balance", "Check the balance at an address", false, NewBalanceCommand, *NewOptionalCommandArg("address", AddressArg)))
	cs.AddCommand(NewCommandDeclaration("connect", "Connect to an RPC endpoint", false, NewConnectCommand, *NewCommandArg("url", StringArg)))
	cs.AddCommand(NewCommandDeclaration("close", "Close the currently open wallet", false, NewCloseCommand))
	cs.AddCommand(NewCommandDeclaration("lock", "Close the currently open wallet", true, NewCloseCommand))
	cs.AddCommand(NewCommandDeclaration("create", "Create and open a new wallet file", false, NewCreateCommand, *NewCommandArg("filename", StringArg), *NewOptionalCommandArg("password", StringArg)))
	cs.AddCommand(NewCommandDeclaration("disconnect", "Disconnect from RPC endpoint", false, NewDisconnectCommand))
	cs.AddCommand(NewCommandDeclaration("generate", "Generate and display a new private key", false, NewGenerateKeyCommand))
	cs.AddCommand(NewCommandDeclaration("help", "Show help on a given command", false, NewHelpCommand, *NewCommandArg("command", CmdNameArg)))
	cs.AddCommand(NewCommandDeclaration("import", "Import a WIF private key to a new wallet file", false, NewImportCommand, *NewCommandArg("private-key", StringArg), *NewCommandArg("filename", StringArg), *NewOptionalCommandArg("password", StringArg)))
	cs.AddCommand(NewCommandDeclaration("list", "List available commands", false, NewListCommand))
	cs.AddCommand(NewCommandDeclaration("upload", "Upload a smart contract", false, NewUploadContractCommand, *NewCommandArg("filename", StringArg)))
	cs.AddCommand(NewCommandDeclaration("call", "Call a smart contract", false, NewCallCommand, *NewCommandArg("contract-id", StringArg), *NewCommandArg("entry-point", StringArg), *NewCommandArg("arguments", StringArg)))
	cs.AddCommand(NewCommandDeclaration("open", "Open a wallet file", false, NewOpenCommand, *NewCommandArg("filename", StringArg), *NewOptionalCommandArg("password", StringArg)))
	cs.AddCommand(NewCommandDeclaration("unlock", "Open a wallet file", true, NewOpenCommand, *NewCommandArg("filename", StringArg), *NewOptionalCommandArg("password", StringArg)))
	cs.AddCommand(NewCommandDeclaration("private", "Show the currently opened wallet's private key", false, NewPrivateCommand))
	cs.AddCommand(NewCommandDeclaration("read", "Read from a smart contract", false, NewReadCommand, *NewCommandArg("contract-id", StringArg), *NewCommandArg("entry-point", StringArg), *NewCommandArg("arguments", StringArg)))
	cs.AddCommand(NewCommandDeclaration("register", "Register a smart contract's commands", false, NewRegisterCommand, *NewCommandArg("name", StringArg), *NewCommandArg("address", AddressArg), *NewCommandArg("abi-filename", StringArg)))
	cs.AddCommand(NewCommandDeclaration("transfer", "Transfer token from an open wallet to a given address", false, NewTransferCommand, *NewCommandArg("amount", AmountArg), *NewCommandArg("address", AddressArg)))
	cs.AddCommand(NewCommandDeclaration("set_system_call", "Set a system call to a new contract and entry point", false, NewSetSystemCallCommand, *NewCommandArg("system-call", StringArg), *NewCommandArg("contract-id", StringArg), *NewCommandArg("entry-point", StringArg)))
	cs.AddCommand(NewCommandDeclaration("exit", "Exit the wallet (quit also works)", false, NewExitCommand))
	cs.AddCommand(NewCommandDeclaration("quit", "", true, NewExitCommand))

	return cs
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
	AddressString *string
}

// NewBalanceCommand creates a new balance object
func NewBalanceCommand(inv *CommandParseResult) CLICommand {
	addressString := inv.Args["address"]
	return &BalanceCommand{AddressString: addressString}
}

// Execute fetches the balance
func (c *BalanceCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot check balance", util.ErrOffline)
	}

	var address []byte
	var err error

	// Get current account balance if empty address
	if c.AddressString == nil {
		if !ee.IsWalletOpen() {
			return nil, fmt.Errorf("%w: must give an address", util.ErrWalletClosed)
		}

		address = ee.Key.AddressBytes()
	} else {
		address = base58.Decode(*c.AddressString)
		if len(address) == 0 {
			return nil, errors.New("could not parse address")
		}
	}

	// Setup command execution environment
	contractID := base58.Decode(KoinContractID)
	if len(contractID) == 0 {
		panic("Invalid KOIN contract ID")
	}

	balance, err := ee.RPCClient.GetAccountBalance(address, contractID, KoinBalanceOfEntry)

	// Build the result
	dec, err := util.SatoshiToDecimal(int64(balance), KoinPrecision)
	if err != nil {
		return nil, err
	}

	// Get Mana
	mana, err := ee.RPCClient.GetAccountRc(address)
	if err != nil {
		return nil, err
	}

	// Build the mana result
	manaDec, err := util.SatoshiToDecimal(int64(mana), KoinPrecision)
	if err != nil {
		return nil, err
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("%v %s", dec, KoinSymbol))
	er.AddMessage(fmt.Sprintf("%v %s", manaDec, ManaSymbol))

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
		return nil, fmt.Errorf("%w: cannot close", util.ErrWalletClosed)
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
	return &ConnectCommand{URL: *inv.Args["url"]}
}

// Execute connects to an RPC endpoint
func (c *ConnectCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	rpc := kjsonrpc.NewKoinosRPCClient(c.URL)
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
		return nil, fmt.Errorf("%w: cannot disconnect", util.ErrOffline)
	}

	// Disconnect from the RPC endpoint
	ee.RPCClient = nil

	result := NewExecutionResult()
	result.AddMessage("Disconnected")

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
	k, err := util.GenerateKoinosKey()
	if err != nil {
		return nil, err
	}

	result := NewExecutionResult()
	result.AddMessage("New key generated. This is only shown once, make sure to record this information.")
	result.AddMessage(fmt.Sprintf("Address: %s", base58.Encode(k.AddressBytes())))
	result.AddMessage(fmt.Sprintf("Private: %s", k.Private()))

	return result, nil
}

// ----------------------------------------------------------------------------
// Upload Contract Command
// ----------------------------------------------------------------------------

// UploadContractCommand is a command that uploads a smart contract
type UploadContractCommand struct {
	Filename string
}

// NewUploadContractCommand creates an upload contract object
func NewUploadContractCommand(inv *CommandParseResult) CLICommand {
	return &UploadContractCommand{Filename: *inv.Args["filename"]}
}

// Execute calls a contract
func (c *UploadContractCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot upload contract", util.ErrWalletClosed)
	}

	// Check if the wallet already exists
	if _, err := os.Stat(c.Filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", util.ErrFileNotFound, c.Filename)
	}

	// Fetch the accounts nonce
	myAddress := ee.Key.AddressBytes()
	nonce, err := ee.RPCClient.GetAccountNonce(myAddress)
	if err != nil {
		return nil, err
	}

	wasmBytes, err := ioutil.ReadFile(c.Filename)

	if err != nil {
		return nil, err
	}

	uc := protocol.Operation_UploadContract{UploadContract: &protocol.UploadContractOperation{ContractId: myAddress, Bytecode: wasmBytes}}
	op := protocol.Operation{Op: &uc}

	rcLimit, err := ee.RPCClient.GetAccountRc(ee.Key.AddressBytes())
	if err != nil {
		return nil, err
	}

	active := protocol.ActiveTransactionData{Nonce: nonce, Operations: []*protocol.Operation{&op}, RcLimit: rcLimit}
	activeBytes, err := canonical.Marshal(&active)
	if err != nil {
		return nil, err
	}

	transaction := protocol.Transaction{Active: activeBytes}

	sha256Hasher := sha256.New()
	sha256Hasher.Write(activeBytes)

	tid, err := multihash.EncodeName(sha256Hasher.Sum(nil), "sha2-256")
	if err != nil {
		return nil, err
	}
	transaction.Id = tid

	err = util.SignTransaction(ee.Key.PrivateBytes(), &transaction)

	if err != nil {
		return nil, err
	}

	params := chain.SubmitTransactionRequest{}
	params.Transaction = &transaction

	// Make the rpc call
	var cResp chain.SubmitTransactionResponse
	err = ee.RPCClient.Call(kjsonrpc.SubmitTransactionCall, &params, &cResp)
	if err != nil {
		return nil, err
	}

	er := NewExecutionResult()
	er.AddMessage(fmt.Sprintf("Contract submitted with ID: %s", base58.Encode(myAddress)))

	return er, nil
}

// ----------------------------------------------------------------------------
// Create
// ----------------------------------------------------------------------------

// CreateCommand is a command that creates a new wallet
type CreateCommand struct {
	Filename string
	Password *string
}

// NewCreateCommand creates a new create object
func NewCreateCommand(inv *CommandParseResult) CLICommand {
	return &CreateCommand{Filename: *inv.Args["filename"], Password: inv.Args["password"]}
}

// Execute creates a new wallet
func (c *CreateCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {

	// Check if the wallet already exists
	if _, err := os.Stat(c.Filename); !os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", util.ErrWalletExists, c.Filename)
	}

	// Generate new key
	key, err := util.GenerateKoinosKey()
	if err != nil {
		return nil, err
	}

	// Create the wallet file
	file, err := os.Create(c.Filename)
	if err != nil {
		return nil, err
	}

	// Get the password
	pass, err := util.GetPassword(c.Password)
	if err != nil {
		return nil, err
	}

	// Write the key to the wallet file
	err = util.CreateWalletFile(file, pass, key.PrivateBytes())
	if err != nil {
		return nil, err
	}

	// Set the wallet keys
	ee.Key = key

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Created and opened new wallet: %s", c.Filename))
	result.AddMessage(fmt.Sprintf("Address: %s", base58.Encode(key.AddressBytes())))

	return result, nil
}

// ----------------------------------------------------------------------------
// Import
// ----------------------------------------------------------------------------

// ImportCommand is a command that imports a private key to a wallet
type ImportCommand struct {
	Filename   string
	Password   *string
	PrivateKey string
}

// NewImportCommand creates a new import object
func NewImportCommand(inv *CommandParseResult) CLICommand {
	return &ImportCommand{Filename: *inv.Args["filename"], Password: inv.Args["password"], PrivateKey: *inv.Args["private-key"]}
}

// Execute creates a new wallet
func (c *ImportCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	// Check if the wallet already exists
	if _, err := os.Stat(c.Filename); !os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", util.ErrWalletExists, c.Filename)
	}

	// Convert the private key to bytes
	keyBytes, err := util.DecodeWIF(c.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Create the key
	key, err := util.NewKoinosKeysFromBytes(keyBytes)
	if err != nil {
		return nil, err
	}

	// Create the wallet file
	file, err := os.Create(c.Filename)
	if err != nil {
		return nil, err
	}

	// Get the password
	pass, err := util.GetPassword(c.Password)
	if err != nil {
		return nil, err
	}

	// Write the key to the wallet file
	err = util.CreateWalletFile(file, pass, key.PrivateBytes())
	if err != nil {
		return nil, err
	}

	// Set the wallet keys
	ee.Key = key

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Created and opened new wallet: %s", c.Filename))
	result.AddMessage(fmt.Sprintf("Address: %s", base58.Encode(key.AddressBytes())))

	return result, nil
}

// ----------------------------------------------------------------------------
// Address Command
// ----------------------------------------------------------------------------

// AddressCommand is a command that shows the currently opened wallet's address and private key
type AddressCommand struct {
}

// NewAddressCommand creates a new address command object
func NewAddressCommand(inv *CommandParseResult) CLICommand {
	return &AddressCommand{}
}

// Execute shows wallet address
func (c *AddressCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot show address", util.ErrWalletClosed)
	}

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Wallet address: %s", base58.Encode(ee.Key.AddressBytes())))

	return result, nil
}

// ----------------------------------------------------------------------------
// Private Command
// ----------------------------------------------------------------------------

// PrivateCommand is a command that shows the currently opened wallet's address and private key
type PrivateCommand struct {
}

// NewPrivateCommand creates a new private command object
func NewPrivateCommand(inv *CommandParseResult) CLICommand {
	return &PrivateCommand{}
}

// Execute shows wallet private key
func (c *PrivateCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot show private key", util.ErrWalletClosed)
	}

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Private key: %s", ee.Key.Private()))

	return result, nil
}

// ----------------------------------------------------------------------------
// Help
// ----------------------------------------------------------------------------

// HelpCommand is a command that displays help for a given command
type HelpCommand struct {
	Command string
}

// NewHelpCommand creates a new help command object
func NewHelpCommand(inv *CommandParseResult) CLICommand {
	return &HelpCommand{Command: *inv.Args["command"]}
}

// Execute displays help for a given command
func (c *HelpCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	decl, ok := ee.Parser.Commands.Name2Command[string(c.Command)]

	if !ok {
		return nil, fmt.Errorf("%w: cannot show help for %s", util.ErrUnknownCommand, c.Command)
	}

	result := NewExecutionResult()
	result.AddMessage(decl.Description)
	result.AddMessage(fmt.Sprintf("Usage: %s", decl))

	return result, nil
}

// ----------------------------------------------------------------------------
// Call Command
// ----------------------------------------------------------------------------

// CallCommand is a command that shows the currently opened wallet's address and private key
type CallCommand struct {
	ContractID string
	EntryPoint string
	Arguments  string
}

// NewCallCommand calls a contract method
func NewCallCommand(inv *CommandParseResult) CLICommand {
	return &CallCommand{
		ContractID: *inv.Args["contract-id"],
		EntryPoint: *inv.Args["entry-point"],
		Arguments:  *inv.Args["arguments"],
	}
}

// Execute a contract call
func (c *CallCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot call contract", util.ErrWalletClosed)
	}

	entryPoint, err := strconv.ParseUint(c.EntryPoint[2:], 16, 32)
	if err != nil {
		return nil, err
	}

	contractID := base58.Decode(c.ContractID)
	if len(contractID) == 0 {
		return nil, errors.New("could not parse contract id")
	}

	// Get the argument bytes
	argumentBytes, err := base64.StdEncoding.DecodeString(c.Arguments[1:])
	if err != nil {
		return nil, err
	}

	_, err = ee.RPCClient.WriteContract(argumentBytes, ee.Key, contractID, uint32(entryPoint))

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Calling contract %s at entry point: %s with arguments %s", c.ContractID, c.EntryPoint, c.Arguments))

	return result, nil
}

// ----------------------------------------------------------------------------
// Open
// ----------------------------------------------------------------------------

// OpenCommand is a command that opens a wallet file
type OpenCommand struct {
	Filename string
	Password *string
}

// NewOpenCommand creates a new open command object
func NewOpenCommand(inv *CommandParseResult) CLICommand {
	return &OpenCommand{Filename: *inv.Args["filename"], Password: inv.Args["password"]}
}

// Execute opens a wallet
func (c *OpenCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	// Open the wallet file
	file, err := os.Open(c.Filename)
	if err != nil {
		return nil, err
	}

	// Get the password
	pass, err := util.GetPassword(c.Password)
	if err != nil {
		return nil, err
	}

	// Read the wallet file
	keyBytes, err := util.ReadWalletFile(file, pass)
	if err != nil {
		return nil, fmt.Errorf("%w: check your password", util.ErrWalletDecrypt)
	}

	// Create the key object
	key, err := util.NewKoinosKeysFromBytes(keyBytes)
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
// Read
// ----------------------------------------------------------------------------

// ReadCommand is a command that reads from a contract
type ReadCommand struct {
	ContractID string
	EntryPoint string
	Arguments  string
}

// NewReadCommand creates a new read command object
func NewReadCommand(inv *CommandParseResult) CLICommand {
	return &ReadCommand{ContractID: *inv.Args["contract-id"], EntryPoint: *inv.Args["entry-point"], Arguments: *inv.Args["arguments"]}
}

// Execute reads from a contract
func (c *ReadCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	cid := base58.Decode(c.ContractID)
	if len(cid) == 0 {
		return nil, errors.New("could not parse contract id")
	}

	// Parse the entry point (drop the 0x)
	entryPoint, err := strconv.ParseUint(c.EntryPoint[2:], 16, 32)
	if err != nil {
		return nil, err
	}

	// Serialize and assign the args
	argumentBytes, err := base64.StdEncoding.DecodeString(c.Arguments[1:])
	if err != nil {
		return nil, err
	}

	cResp, err := ee.RPCClient.ReadContract(argumentBytes, cid, uint32(entryPoint))
	if err != nil {
		return nil, err
	}

	result := NewExecutionResult()
	result.AddMessage("M" + base64.StdEncoding.EncodeToString(cResp.Result))

	return result, nil
}

// ----------------------------------------------------------------------------
// Transfer
// ----------------------------------------------------------------------------

// TransferCommand is a command that closes an open wallet
type TransferCommand struct {
	Address string
	Amount  string
}

// NewTransferCommand creates a new close object
func NewTransferCommand(inv *CommandParseResult) CLICommand {
	addressString := inv.Args["address"]
	return &TransferCommand{Address: *addressString, Amount: *inv.Args["amount"]}
}

// Execute transfers token
func (c *TransferCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot transfer", util.ErrWalletClosed)
	}

	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot transfer", util.ErrOffline)
	}

	// Convert the amount to a decimal
	dAmount, err := decimal.NewFromString(c.Amount)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", util.ErrInvalidAmount, err.Error())
	}

	// Convert the amount to satoshis
	sAmount, err := util.DecimalToSatoshi(&dAmount, KoinPrecision)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", util.ErrInvalidAmount, err.Error())
	}

	// Ensure a transfer greater than zero
	if sAmount <= 0 {
		minimalAmount, _ := util.SatoshiToDecimal(1, KoinPrecision)
		return nil, fmt.Errorf("%w: cannot transfer %s %s, amount should be greater than minimal %s (1e-%d) %s", util.ErrInvalidAmount, dAmount, KoinSymbol, minimalAmount, KoinPrecision, KoinSymbol)
	}

	// Setup command execution environment
	contractID := base58.Decode(KoinContractID)
	if len(contractID) == 0 {
		panic("Invalid KOIN contract ID")
	}

	// Fetch the account's balance
	myAddress := ee.Key.AddressBytes()
	balance, err := ee.RPCClient.GetAccountBalance(myAddress, contractID, KoinBalanceOfEntry)
	if err != nil {
		return nil, err
	}
	dBalance, err := util.SatoshiToDecimal(int64(balance), KoinPrecision)
	if err != nil {
		return nil, err
	}

	// Ensure a transfer greater than opened account balance
	if int64(balance) <= sAmount {
		return nil, fmt.Errorf("%w: insufficient balance %s %s on opened wallet %s, cannot transfer %s %s", util.ErrInvalidAmount, dBalance, KoinSymbol, myAddress, dAmount, KoinSymbol)
	}

	toAddress := base58.Decode(c.Address)
	if len(toAddress) == 0 {
		return nil, errors.New("could not parse address")
	}

	transferArgs := &token.TransferArguments{
		From:  myAddress,
		To:    toAddress,
		Value: uint64(sAmount),
	}

	// Execute the transfer
	_, err = ee.RPCClient.WriteMessageContract(transferArgs, ee.Key, contractID, KoinTransferEntry)
	if err != nil {
		return nil, err
	}

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Transferring %s %s to %s", dAmount, KoinSymbol, c.Address))

	return result, nil
}

// ----------------------------------------------------------------------------
// SetSystemCall Command
// ----------------------------------------------------------------------------

// SetSystemCallCommand is a command that sets a system call to a new contract and entry point
type SetSystemCallCommand struct {
	SystemCall string
	ContractID string
	EntryPoint string
}

// NewSetSystemCallCommand calls a contract method
func NewSetSystemCallCommand(inv *CommandParseResult) CLICommand {
	return &SetSystemCallCommand{
		SystemCall: *inv.Args["system-call"],
		ContractID: *inv.Args["contract-id"],
		EntryPoint: *inv.Args["entry-point"],
	}
}

// Execute a contract call
func (c *SetSystemCallCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot call contract", util.ErrWalletClosed)
	}

	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot call contract", util.ErrOffline)
	}

	systemCall, err := strconv.ParseUint(c.SystemCall, 10, 32)
	if err != nil {
		if sysCall, ok := protocol.SystemCallId_value[c.SystemCall]; ok {
			systemCall = uint64(sysCall)
		} else {
			return nil, fmt.Errorf("no system call: %s", c.SystemCall)
		}
	}

	entryPoint, err := strconv.ParseUint(c.EntryPoint[2:], 16, 32)
	if err != nil {
		return nil, err
	}

	contractID := base58.Decode(c.ContractID)
	if len(contractID) == 0 {
		return nil, errors.New("could not parse contract id")
	}

	_, err = ee.RPCClient.SetSystemCall(uint32(systemCall), ee.Key, contractID, uint32(entryPoint))

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Setting system call %s to contract %s at entry point %s", c.SystemCall, c.ContractID, c.EntryPoint))

	return result, nil
}

// ----------------------------------------------------------------------------
// List
// ----------------------------------------------------------------------------

// ListCommand is a command that lists available commands
type ListCommand struct {
}

// NewListCommand creates a new list command object
func NewListCommand(inv *CommandParseResult) CLICommand {
	return &ListCommand{}
}

// Execute lists available commands
func (c *ListCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	cmds := ee.Parser.Commands.List(true)

	result := NewExecutionResult()
	result.AddMessage(cmds...)

	return result, nil
}
