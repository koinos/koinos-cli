package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/koinos/koinos-cli/internal/cliutil"
	kjson "github.com/koinos/koinos-proto-golang/encoding/json"
	"github.com/koinos/koinos-proto-golang/koinos/chain"
	"github.com/koinos/koinos-proto-golang/koinos/protocol"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	util "github.com/koinos/koinos-util-golang"
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
	cs.AddCommand(NewCommandDeclaration("connect", "Connect to an RPC endpoint", false, NewConnectCommand, *NewCommandArg("url", StringArg)))
	cs.AddCommand(NewCommandDeclaration("close", "Close the currently open wallet (lock also works)", false, NewCloseCommand))
	cs.AddCommand(NewCommandDeclaration("lock", "Synonym for close", true, NewCloseCommand))
	cs.AddCommand(NewCommandDeclaration("create", "Create and open a new wallet file", false, NewCreateCommand, *NewCommandArg("filename", FileArg), *NewOptionalCommandArg("password", StringArg)))
	cs.AddCommand(NewCommandDeclaration("disconnect", "Disconnect from RPC endpoint", false, NewDisconnectCommand))
	cs.AddCommand(NewCommandDeclaration("generate", "Generate and display a new private key", false, NewGenerateKeyCommand))
	cs.AddCommand(NewCommandDeclaration("help", "Show help on a given command", false, NewHelpCommand, *NewCommandArg("command", CmdNameArg)))
	cs.AddCommand(NewCommandDeclaration("import", "Import a WIF private key to a new wallet file", false, NewImportCommand, *NewCommandArg("private-key", StringArg), *NewCommandArg("filename", FileArg), *NewOptionalCommandArg("password", StringArg)))
	cs.AddCommand(NewCommandDeclaration("list", "List available commands", false, NewListCommand))
	cs.AddCommand(NewCommandDeclaration("upload", "Upload a smart contract", false, NewUploadContractCommand, *NewCommandArg("filename", FileArg), *NewOptionalCommandArg("abi-filename", FileArg), *NewOptionalCommandArg("override-authorize-call-contract", BoolArg), *NewOptionalCommandArg("override-authorize-transaction-application", BoolArg), *NewOptionalCommandArg("override-authorize-upload-contract", BoolArg)))
	cs.AddCommand(NewCommandDeclaration("call", "Call a smart contract", false, NewCallCommand, *NewCommandArg("contract-id", StringArg), *NewCommandArg("entry-point", HexArg), *NewCommandArg("arguments", StringArg)))
	cs.AddCommand(NewCommandDeclaration("open", "Open a wallet file (unlock also works)", false, NewOpenCommand, *NewCommandArg("filename", FileArg), *NewOptionalCommandArg("password", StringArg)))
	cs.AddCommand(NewCommandDeclaration("unlock", "Synonym for open", true, NewOpenCommand, *NewCommandArg("filename", FileArg), *NewOptionalCommandArg("password", StringArg)))
	cs.AddCommand(NewCommandDeclaration("nonce", "Set nonce for transactions. 'auto' will default to querying for nonce. Blank nonce to view", false, NewNonceCommand, *NewOptionalCommandArg("nonce", StringArg)))
	cs.AddCommand(NewCommandDeclaration("chain_id", "Set chain id in base64 for transactions. 'auto' will default to querying for chain id. Blank id to view", false, NewChainIDCommand, *NewOptionalCommandArg("id", StringArg)))
	cs.AddCommand(NewCommandDeclaration("payer", "Set the payer address for transactions. 'me' will default to current wallet. Blank address to view", false, NewPayerCommand, *NewOptionalCommandArg("payer", AddressArg)))
	cs.AddCommand(NewCommandDeclaration("private", "Show the currently opened wallet's private key", false, NewPrivateCommand))
	cs.AddCommand(NewCommandDeclaration("public", "Show the currently opened wallet's public key", false, NewPublicCommand))
	cs.AddCommand(NewCommandDeclaration("rclimit", "Set or show the current rc limit. Give no limit to see current value. Give limit as either mana or a percent (i.e. 80%).", false, NewRcLimitCommand, *NewOptionalCommandArg("limit", StringArg)))
	cs.AddCommand(NewCommandDeclaration("read", "Read from a smart contract", false, NewReadCommand, *NewCommandArg("contract-id", StringArg), *NewCommandArg("entry-point", StringArg), *NewCommandArg("arguments", StringArg)))
	cs.AddCommand(NewCommandDeclaration("register", "Register a smart contract's commands", false, NewRegisterCommand, *NewCommandArg("name", ContractNameArg), *NewCommandArg("address", AddressArg), *NewOptionalCommandArg("abi-filename", FileArg)))
	cs.AddCommand(NewCommandDeclaration("register_token", "Register a token's commands", false, NewRegisterTokenCommand, *NewCommandArg("name", ContractNameArg), *NewCommandArg("address", AddressArg), *NewOptionalCommandArg("symbol", StringArg), *NewOptionalCommandArg("precision", StringArg)))
	cs.AddCommand(NewCommandDeclaration("account_rc", "Get the current resource credits for a given address (open wallet if blank)", false, NewAccountRcCommand, *NewOptionalCommandArg("address", AddressArg)))
	cs.AddCommand(NewCommandDeclaration("account_nonce", "Get the current nonce for a given address (open wallet if blank)", false, NewAccountNonceCommand, *NewOptionalCommandArg("address", AddressArg)))
	cs.AddCommand(NewCommandDeclaration("set_system_call", "Set a system call to a new contract and entry point", false, NewSetSystemCallCommand, *NewCommandArg("system-call", StringArg), *NewCommandArg("contract-id", AddressArg), *NewCommandArg("entry-point", HexArg)))
	cs.AddCommand(NewCommandDeclaration("set_system_contract", "Change a contract's permission level between user and system", false, NewSetSystemContractCommand, *NewCommandArg("contract-id", AddressArg), *NewCommandArg("system-contract", BoolArg)))
	cs.AddCommand(NewCommandDeclaration("session", "Create or manage a transaction session (begin, submit, cancel, or view)", false, NewSessionCommand, *NewCommandArg("command", StringArg)))
	cs.AddCommand(NewCommandDeclaration("sign_transaction", "Signs a transaction with the open wallet, adding it to the transaction", true, NewSignTransactionCommand, *NewCommandArg("transaction", StringArg)))
	cs.AddCommand(NewCommandDeclaration("submit_transaction", "Submit a transaction from base64 data", false, NewSubmitTransactionCommand, *NewCommandArg("transaction", StringArg)))
	cs.AddCommand(NewCommandDeclaration("sleep", "Sleep for the given number seconds", true, NewSleepCommand, *NewCommandArg("seconds", AmountArg)))
	cs.AddCommand(NewCommandDeclaration("exit", "Exit the wallet (quit also works)", false, NewExitCommand))
	cs.AddCommand(NewCommandDeclaration("quit", "Synonym for exit", true, NewExitCommand))

	return cs
}

// ----------------------------------------------------------------------------
// Command Implementations
// ----------------------------------------------------------------------------

// All commands should be implemented here

// ----------------------------------------------------------------------------
// Close Command
// ----------------------------------------------------------------------------

// CloseCommand is a command that closes an open wallet
type CloseCommand struct {
}

// NewCloseCommand creates a new close object
func NewCloseCommand(inv *CommandParseResult) Command {
	return &CloseCommand{}
}

// Execute closes the wallet
func (c *CloseCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot close", cliutil.ErrWalletClosed)
	}

	// Close the wallet
	ee.CloseWallet()

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
func NewConnectCommand(inv *CommandParseResult) Command {
	return &ConnectCommand{URL: *inv.Args["url"]}
}

// Execute connects to an RPC endpoint
func (c *ConnectCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	rpc := cliutil.NewKoinosRPCClient(c.URL)
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
func NewDisconnectCommand(inv *CommandParseResult) Command {
	return &DisconnectCommand{}
}

// Execute disconnects from an RPC endpoint
func (c *DisconnectCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot disconnect", cliutil.ErrOffline)
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

// ExitCommand is a command that exits the CLI
type ExitCommand struct {
}

// NewExitCommand creates a new exit object
func NewExitCommand(inv *CommandParseResult) Command {
	return &ExitCommand{}
}

// Execute exits the CLI
func (c *ExitCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	os.Exit(0)
	return nil, nil
}

// ----------------------------------------------------------------------------
// Generate Key Command
// ----------------------------------------------------------------------------

// GenerateKeyCommand is a command that generates anonymous keys
type GenerateKeyCommand struct {
}

// NewGenerateKeyCommand creates a new exit object
func NewGenerateKeyCommand(inv *CommandParseResult) Command {
	return &GenerateKeyCommand{}
}

// Execute generates anonymous keys
func (c *GenerateKeyCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	k, err := util.GenerateKoinosKey()
	if err != nil {
		return nil, err
	}

	result := NewExecutionResult()
	result.AddMessage("New key generated\nThis is only shown once, make sure to record this information\n---")
	result.AddMessage(fmt.Sprintf("Address: %s", base58.Encode(k.AddressBytes())))
	result.AddMessage(fmt.Sprintf("Public : %s", base64.URLEncoding.EncodeToString(k.PublicBytes())))
	result.AddMessage(fmt.Sprintf("Private: %s", k.Private()))

	return result, nil
}

// ----------------------------------------------------------------------------
// Upload Contract Command
// ----------------------------------------------------------------------------

// UploadContractCommand is a command that uploads a smart contract
type UploadContractCommand struct {
	Filename                         string
	ABIFilename                      *string
	AuthorizesCallContract           *string
	AuthorizesTransactionApplication *string
	AuthorizesUploadContract         *string
}

// NewUploadContractCommand creates an upload contract object
func NewUploadContractCommand(inv *CommandParseResult) Command {
	return &UploadContractCommand{
		Filename:                         *inv.Args["filename"],
		ABIFilename:                      inv.Args["abi-filename"],
		AuthorizesCallContract:           inv.Args["override-authorize-call-contract"],
		AuthorizesTransactionApplication: inv.Args["override-authorize-transaction-application"],
		AuthorizesUploadContract:         inv.Args["override-authorize-upload-contract"],
	}
}

// Execute uploads a contract
func (c *UploadContractCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot upload contract", cliutil.ErrWalletClosed)
	}

	if !ee.IsOnline() && !ee.Session.IsValid() {
		return nil, fmt.Errorf("%w: cannot upload contract", cliutil.ErrOffline)
	}

	// Check if the wallet already exists
	if _, err := os.Stat(c.Filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", cliutil.ErrFileNotFound, c.Filename)
	}

	wasmBytes, err := ioutil.ReadFile(c.Filename)
	if err != nil {
		return nil, err
	}

	// Make the upload contract operation
	uco := &protocol.UploadContractOperation{
		ContractId: ee.Key.AddressBytes(),
		Bytecode:   wasmBytes,
	}

	// Load the ABI if given
	if c.ABIFilename != nil {
		abiFile, err := os.Open(*c.ABIFilename)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, err)
		}

		defer abiFile.Close()

		abiBytes, err := ioutil.ReadAll(abiFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, err)
		}

		// Do a sanity check to make sure the abi file deserializes properly
		var abi ABI
		err = json.Unmarshal(abiBytes, &abi)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", cliutil.ErrInvalidABI, err)
		}

		uco.Abi = string(abiBytes)
	}

	// parse AuthorizesCallContract if given
	if c.AuthorizesCallContract != nil {
		authorizesCallContract, err := strconv.ParseBool(*c.AuthorizesCallContract)
		if err != nil {
			return nil, err
		}

		uco.AuthorizesCallContract = authorizesCallContract
	}

	// parse AuthorizesTransactionApplication if given
	if c.AuthorizesTransactionApplication != nil {
		authorizesTransactionApplication, err := strconv.ParseBool(*c.AuthorizesTransactionApplication)
		if err != nil {
			return nil, err
		}

		uco.AuthorizesTransactionApplication = authorizesTransactionApplication
	}

	// parse AuthorizesUploadContract if given
	if c.AuthorizesUploadContract != nil {
		authorizesUploadContract, err := strconv.ParseBool(*c.AuthorizesUploadContract)
		if err != nil {
			return nil, err
		}

		uco.AuthorizesUploadContract = authorizesUploadContract
	}

	// Make the operation object
	op := &protocol.Operation{
		Op: &protocol.Operation_UploadContract{
			UploadContract: uco,
		},
	}

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Contract uploaded with address %s", base58.Encode(ee.Key.AddressBytes())))

	err = ee.Session.AddOperation(op, fmt.Sprintf("Upload contract with address %s", base58.Encode(ee.Key.AddressBytes())))
	if err == nil {
		result.AddMessage("Adding operation to transaction session")
	}
	if err != nil {
		err := ee.SubmitTransaction(ctx, result, op)
		if err != nil {
			return result, fmt.Errorf("cannot upload contract, %w", err)
		}
	}

	return result, nil
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
func NewCreateCommand(inv *CommandParseResult) Command {
	return &CreateCommand{Filename: *inv.Args["filename"], Password: inv.Args["password"]}
}

// Execute creates a new wallet
func (c *CreateCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {

	// Check if the wallet already exists
	if _, err := os.Stat(c.Filename); !os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", cliutil.ErrWalletExists, c.Filename)
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
	pass, err := cliutil.GetPassword(c.Password)
	if err != nil {
		return nil, err
	}

	// Write the key to the wallet file
	err = cliutil.CreateWalletFile(file, pass, key.PrivateBytes())
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
func NewImportCommand(inv *CommandParseResult) Command {
	return &ImportCommand{Filename: *inv.Args["filename"], Password: inv.Args["password"], PrivateKey: *inv.Args["private-key"]}
}

// Execute creates a new wallet
func (c *ImportCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	// Check if the wallet already exists
	if _, err := os.Stat(c.Filename); !os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", cliutil.ErrWalletExists, c.Filename)
	}

	// Convert the private key to bytes
	keyBytes, err := util.DecodeWIF(c.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Create the key
	key, err := util.NewKoinosKeyFromBytes(keyBytes)
	if err != nil {
		return nil, err
	}

	// Create the wallet file
	file, err := os.Create(c.Filename)
	if err != nil {
		return nil, err
	}

	// Get the password
	pass, err := cliutil.GetPassword(c.Password)
	if err != nil {
		return nil, err
	}

	// Write the key to the wallet file
	err = cliutil.CreateWalletFile(file, pass, key.PrivateBytes())
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
func NewAddressCommand(inv *CommandParseResult) Command {
	return &AddressCommand{}
}

// Execute shows wallet address
func (c *AddressCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot show address", cliutil.ErrWalletClosed)
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
func NewPrivateCommand(inv *CommandParseResult) Command {
	return &PrivateCommand{}
}

// Execute shows wallet private key
func (c *PrivateCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot show private key", cliutil.ErrWalletClosed)
	}

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Private key: %s", ee.Key.Private()))

	return result, nil
}

// ----------------------------------------------------------------------------
// Public Command
// ----------------------------------------------------------------------------

// PublicCommand is a command that shows the currently opened wallet's public
type PublicCommand struct {
}

// NewPublicCommand creates a new public command object
func NewPublicCommand(inv *CommandParseResult) Command {
	return &PublicCommand{}
}

// Execute shows wallet public key
func (c *PublicCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot show public key", cliutil.ErrWalletClosed)
	}

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Public key: %s", base64.URLEncoding.EncodeToString(ee.Key.PublicBytes())))

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
func NewHelpCommand(inv *CommandParseResult) Command {
	return &HelpCommand{Command: *inv.Args["command"]}
}

// Execute displays help for a given command
func (c *HelpCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	decl, ok := ee.Parser.Commands.Name2Command[string(c.Command)]

	if !ok {
		return nil, fmt.Errorf("%w: cannot show help for %s", cliutil.ErrUnknownCommand, c.Command)
	}

	result := NewExecutionResult()
	result.AddMessage(decl.Description)
	result.AddMessage(fmt.Sprintf("Usage: %s", decl))

	return result, nil
}

// ----------------------------------------------------------------------------
// Submit Transaction Command
// ----------------------------------------------------------------------------

// SubmitTransactionCommand is a command that submits a given transaction to the blockchain
type SubmitTransactionCommand struct {
	Transaction string
}

// NewSubmitTransactionCommand creates a new submit transaction command object
func NewSubmitTransactionCommand(inv *CommandParseResult) Command {
	return &SubmitTransactionCommand{Transaction: *inv.Args["transaction"]}
}

// Execute submits a transaction to the blockchain
func (c *SubmitTransactionCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	result := NewExecutionResult()

	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot submit transaction", cliutil.ErrOffline)
	}

	// Decode the transaction
	data, err := base64.URLEncoding.DecodeString(c.Transaction)
	if err != nil {
		return nil, err
	}

	transaction := &protocol.Transaction{}
	err = proto.Unmarshal(data, transaction)
	if err != nil {
		return nil, err
	}

	receipt, err := ee.RPCClient.SubmitTransaction(ctx, transaction, true)
	if err != nil {
		return result, err
	}

	result.AddMessage(cliutil.TransactionReceiptToString(receipt, len(transaction.GetOperations())))

	return result, nil
}

// ----------------------------------------------------------------------------
// Call Command
// ----------------------------------------------------------------------------

// CallCommand is a command that calls a contract method
type CallCommand struct {
	ContractID string
	EntryPoint string
	Arguments  string
}

// NewCallCommand calls a contract method
func NewCallCommand(inv *CommandParseResult) Command {
	return &CallCommand{
		ContractID: *inv.Args["contract-id"],
		EntryPoint: *inv.Args["entry-point"],
		Arguments:  *inv.Args["arguments"],
	}
}

// Execute a contract call
func (c *CallCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot call contract", cliutil.ErrWalletClosed)
	}

	if !ee.IsOnline() && !ee.Session.IsValid() {
		return nil, fmt.Errorf("%w: cannot call", cliutil.ErrOffline)
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
	argumentBytes, err := base64.StdEncoding.DecodeString(c.Arguments)
	if err != nil {
		return nil, err
	}

	op := &protocol.Operation{
		Op: &protocol.Operation_CallContract{
			CallContract: &protocol.CallContractOperation{
				ContractId: contractID,
				EntryPoint: uint32(entryPoint),
				Args:       argumentBytes,
			},
		},
	}

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Calling contract %s at entry point: %s with arguments %s", c.ContractID, c.EntryPoint, c.Arguments))

	err = ee.Session.AddOperation(op, fmt.Sprintf("Call contract %s at entry point: %s with arguments %s", c.ContractID, c.EntryPoint, c.Arguments))
	if err == nil {
		result.AddMessage("Adding operation to transaction session")
	}
	if err != nil {
		err := ee.SubmitTransaction(ctx, result, op)
		if err != nil {
			return result, fmt.Errorf("cannot call contract, %w", err)
		}
	}

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
func NewOpenCommand(inv *CommandParseResult) Command {
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
	pass, err := cliutil.GetPassword(c.Password)
	if err != nil {
		return nil, err
	}

	// Read the wallet file
	keyBytes, err := cliutil.ReadWalletFile(file, pass)
	if err != nil {
		return nil, fmt.Errorf("%w: check your password", cliutil.ErrWalletDecrypt)
	}

	// Create the key object
	key, err := util.NewKoinosKeyFromBytes(keyBytes)
	if err != nil {
		return nil, err
	}

	// Open the wallet
	ee.OpenWallet(key)

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Opened wallet: %s", c.Filename))

	return result, nil
}

// ----------------------------------------------------------------------------
// Payer Command
// ----------------------------------------------------------------------------

// PayerCommand is a command shows or sets the current payer
type PayerCommand struct {
	Payer *string
}

// NewPayerCommand creates a new payer command object
func NewPayerCommand(inv *CommandParseResult) Command {
	payerString := inv.Args["payer"]
	return &PayerCommand{Payer: payerString}
}

// Execute shows wallet address
func (c *PayerCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	result := NewExecutionResult()

	// If the payer string is null, then we are showing the current payer
	if c.Payer == nil {
		if ee.IsSelfPaying() {
			if ee.IsWalletOpen() {
				result.AddMessage(fmt.Sprintf("Payer: me (%s)", base58.Encode(ee.GetPayerAddress())))
			} else {
				result.AddMessage("Payer: me")
			}
		} else {
			result.AddMessage(fmt.Sprintf("Payer: %s", base58.Encode(ee.GetPayerAddress())))
		}

		return result, nil
	}

	// Otherwise, we are setting the payer
	ee.SetPayer(*c.Payer)
	return result, nil
}

// ----------------------------------------------------------------------------
// Nonce Command
// ----------------------------------------------------------------------------

// NonceCommand is a command that shows or sets the current nonce
type NonceCommand struct {
	Nonce *string
}

// NewNonceCommand creates a new nonce command object
func NewNonceCommand(inv *CommandParseResult) Command {
	nonceString := inv.Args["nonce"]
	return &NonceCommand{Nonce: nonceString}
}

// Execute shows or sets the current nonce
func (c *NonceCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	result := NewExecutionResult()

	// If the nonce string is null, then we are showing the current nonce
	if c.Nonce == nil {
		if ee.IsNonceAuto() {
			if ee.IsOnline() && ee.IsWalletOpen() {
				nonce, err := ee.GetNextNonce(ctx, false)
				if err != nil {
					return nil, err
				}
				result.AddMessage(fmt.Sprintf("Nonce: auto (next nonce: %d)", nonce))
			} else {
				result.AddMessage("Nonce: auto")
			}
		} else {
			n, err := ee.GetNextNonce(ctx, false)
			if err != nil {
				return nil, err
			}
			result.AddMessage(fmt.Sprintf("Nonce: %d", n))
		}

		return result, nil
	}

	// Otherwise, we are setting the nonce

	// If it's auto just set that
	if *c.Nonce == AutoNonce {
		ee.nonceMode = AutoNonce
		return result, nil
	}

	// Otherwise, parse the nonce to make sure it is correct
	_, err := strconv.ParseUint(*c.Nonce, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: nonce must either be an integer number or \"auto\"", cliutil.ErrInvalidParam)
	}

	ee.nonceMode = *c.Nonce

	return result, nil
}

// ----------------------------------------------------------------------------
// ChainID Command
// ----------------------------------------------------------------------------

// ChainIDCommand is a command that shows or sets the current chain ID
type ChainIDCommand struct {
	ID *string
}

// NewChainIDCommand creates a new chain ID command object
func NewChainIDCommand(inv *CommandParseResult) Command {
	nonceString := inv.Args["id"]
	return &ChainIDCommand{ID: nonceString}
}

// Execute shows or sets the current chain ID
func (c *ChainIDCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	result := NewExecutionResult()

	// If the id string is null, then we are showing the current chain id
	if c.ID == nil {
		if ee.IsChainIDAuto() && ee.IsOnline() {
			chainID, err := ee.GetChainID(ctx)
			if err != nil {
				return nil, err
			}
			result.AddMessage(fmt.Sprintf("Chain ID: auto (%s)", base64.URLEncoding.EncodeToString(chainID)))
		} else {
			result.AddMessage(fmt.Sprintf("Chain ID: %s", ee.chainID))
		}
		return result, nil
	}

	// Otherwise, we are setting the chain id

	// If it's auto just set that
	if *c.ID == AutoChainID {
		ee.chainID = AutoChainID
		return result, nil
	}

	// Make sure the chain id is valid base64
	_, err := base64.URLEncoding.DecodeString(*c.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: chain id must either be a base64 string or \"auto\"", cliutil.ErrInvalidParam)
	}

	ee.chainID = *c.ID

	return result, nil
}

// ----------------------------------------------------------------------------
// RcLimit Command
// ----------------------------------------------------------------------------

// RcLimitCommand is a command that sets or checks your cuttent rc limit
type RcLimitCommand struct {
	limit *string
}

// NewRcLimitCommand creates a new rc limit command object
func NewRcLimitCommand(inv *CommandParseResult) Command {
	return &RcLimitCommand{limit: inv.Args["limit"]}
}

// Execute handles the rc limit command
func (c *RcLimitCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	result := NewExecutionResult()
	// If no limit given, display current
	if c.limit == nil {
		if ee.rcLimit.absolute {
			decAmount, err := util.SatoshiToDecimal(ee.rcLimit.value, cliutil.KoinPrecision)
			if err != nil {
				return nil, err
			}
			result.AddMessage(fmt.Sprintf("Current rc limit: %v", decAmount))
			return result, nil
		}

		// Otherwise its relative
		if !ee.IsOnline() || !ee.IsWalletOpen() {
			decAmount, err := util.SatoshiToDecimal(ee.rcLimit.value, cliutil.KoinPrecision)
			resultVal := decimal.NewFromFloat(100).Mul(*decAmount)
			if err != nil {
				return nil, err
			}
			result.AddMessage(fmt.Sprintf("Current rc limit: %v%%", resultVal))
			return result, nil
		}

		amount, err := ee.GetRcLimit(ctx)
		if err != nil {
			return nil, err
		}

		decAmount, err := util.SatoshiToDecimal(amount, cliutil.KoinPrecision)
		if err != nil {
			return nil, err
		}

		decLimit, err := util.SatoshiToDecimal(ee.rcLimit.value, cliutil.KoinPrecision)
		if err != nil {
			return nil, err
		}

		result.AddMessage(fmt.Sprintf("Current rc limit: %v%% (%v)", decLimit.Mul(decimal.NewFromInt(100)), decAmount))
		return result, nil
	}

	// Otherwise we are setting the limit
	s := *c.limit
	if s[len(s)-1] == '%' {
		res, err := decimal.NewFromString(s[:len(s)-1])
		if err != nil {
			return nil, err
		}

		// Check bounds
		if res.LessThan(decimal.NewFromInt(0)) || res.GreaterThan(decimal.NewFromInt(100)) {
			return nil, fmt.Errorf("%w: percentage rc limit must be between 0%% and 100%%", cliutil.ErrInvalidParam)
		}

		// Convert to decimal
		resFrac := res.Div(decimal.NewFromInt(100))
		val, err := util.DecimalToSatoshi(&resFrac, cliutil.KoinPrecision)
		if err != nil {
			return nil, err
		}

		ee.rcLimit.value = val
		ee.rcLimit.absolute = false
		result.AddMessage(fmt.Sprintf("Set rc limit to %v%%", res))
		return result, nil
	}

	// Otherwise we are setting the absolute limit
	res, err := decimal.NewFromString(s)
	if err != nil {
		return nil, err
	}

	// Convert to satoshi
	val, err := util.DecimalToSatoshi(&res, cliutil.KoinPrecision)
	if err != nil {
		return nil, err
	}

	ee.rcLimit.value = val
	ee.rcLimit.absolute = true
	result.AddMessage(fmt.Sprintf("Set rc limit to %v", res))

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
func NewReadCommand(inv *CommandParseResult) Command {
	return &ReadCommand{ContractID: *inv.Args["contract-id"], EntryPoint: *inv.Args["entry-point"], Arguments: *inv.Args["arguments"]}
}

// Execute reads from a contract
func (c *ReadCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot read contract", cliutil.ErrOffline)
	}

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

	cResp, err := ee.RPCClient.ReadContract(ctx, argumentBytes, cid, uint32(entryPoint))
	if err != nil {
		return nil, err
	}

	result := NewExecutionResult()
	result.AddMessage("M" + base64.StdEncoding.EncodeToString(cResp.Result))

	return result, nil
}

// ----------------------------------------------------------------------------
// Sleep Command
// ----------------------------------------------------------------------------

// SleepCommand is a command that shows the currently opened wallet's address and private key
type SleepCommand struct {
	Duration time.Duration
}

// NewSleepCommand creates a new address command object
func NewSleepCommand(inv *CommandParseResult) Command {
	f, err := strconv.ParseFloat(*inv.Args["seconds"], 32)
	if err != nil {
		return nil
	}

	return &SleepCommand{Duration: time.Duration(f * float64(time.Second))}
}

// Execute shows wallet address
func (c *SleepCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Slept for %s", c.Duration))
	time.Sleep(c.Duration)

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
func NewSetSystemCallCommand(inv *CommandParseResult) Command {
	return &SetSystemCallCommand{
		SystemCall: *inv.Args["system-call"],
		ContractID: *inv.Args["contract-id"],
		EntryPoint: *inv.Args["entry-point"],
	}
}

// Execute a contract call
func (c *SetSystemCallCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot call contract", cliutil.ErrWalletClosed)
	}

	if !ee.IsOnline() && !ee.Session.IsValid() {
		return nil, fmt.Errorf("%w: cannot call contract", cliutil.ErrOffline)
	}

	systemCall, err := strconv.ParseUint(c.SystemCall, 10, 32)
	if err != nil {
		if sysCall, ok := chain.SystemCallId_value[c.SystemCall]; ok {
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

	op := &protocol.Operation{
		Op: &protocol.Operation_SetSystemCall{
			SetSystemCall: &protocol.SetSystemCallOperation{
				CallId: uint32(systemCall),
				Target: &protocol.SystemCallTarget{
					Target: &protocol.SystemCallTarget_SystemCallBundle{
						SystemCallBundle: &protocol.ContractCallBundle{
							ContractId: contractID,
							EntryPoint: uint32(entryPoint),
						},
					},
				},
			},
		},
	}

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Setting system call %s to contract %s at entry point %s", c.SystemCall, c.ContractID, c.EntryPoint))

	err = ee.Session.AddOperation(op, fmt.Sprintf("Set system call %s to contract %s at entry point %s", c.SystemCall, c.ContractID, c.EntryPoint))
	if err == nil {
		result.AddMessage("Adding operation to transaction session")
	}
	if err != nil {
		err := ee.SubmitTransaction(ctx, result, op)
		if err != nil {
			return result, fmt.Errorf("cannot set system call, %w", err)
		}
	}

	return result, nil
}

// ----------------------------------------------------------------------------
// SetSystemContract Command
// ----------------------------------------------------------------------------

// SetSystemContractCommand is a command that sets a system call to a new contract and entry point
type SetSystemContractCommand struct {
	ContractID     string
	SystemContract string
}

// NewSetSystemContractCommand calls a contract method
func NewSetSystemContractCommand(inv *CommandParseResult) Command {
	return &SetSystemContractCommand{
		ContractID:     *inv.Args["contract-id"],
		SystemContract: *inv.Args["system-contract"],
	}
}

// Execute a contract call
func (c *SetSystemContractCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot set system contract", cliutil.ErrWalletClosed)
	}

	if !ee.IsOnline() && !ee.Session.IsValid() {
		return nil, fmt.Errorf("%w: cannot set system contract", cliutil.ErrOffline)
	}

	contractID := base58.Decode(c.ContractID)
	if len(contractID) == 0 {
		return nil, errors.New("could not parse contract id")
	}

	systemContract, err := strconv.ParseBool(c.SystemContract)
	if err != nil {
		return nil, err
	}

	op := &protocol.Operation{
		Op: &protocol.Operation_SetSystemContract{
			SetSystemContract: &protocol.SetSystemContractOperation{
				ContractId:     contractID,
				SystemContract: systemContract,
			},
		},
	}

	result := NewExecutionResult()
	if systemContract {
		result.AddMessage(fmt.Sprintf("Setting contract %s to system level permissions", c.ContractID))
		err = ee.Session.AddOperation(op, fmt.Sprintf("Setting contract %s to system level permissions", c.ContractID))
	} else {
		result.AddMessage(fmt.Sprintf("Setting contract %s to user level permissions", c.ContractID))
		err = ee.Session.AddOperation(op, fmt.Sprintf("Setting contract %s to user level permissions", c.ContractID))
	}

	if err == nil {
		result.AddMessage("Adding operation to transaction session")
	}
	if err != nil {
		err := ee.SubmitTransaction(ctx, result, op)
		if err != nil {
			return result, fmt.Errorf("cannot set contract, %w", err)
		}
	}

	return result, nil
}

// ----------------------------------------------------------------------------
// Session Command
// ----------------------------------------------------------------------------

// SessionCommand is a command that sets a system call to a new contract and entry point
type SessionCommand struct {
	Command string
}

// NewSessionCommand calls a contract method
func NewSessionCommand(inv *CommandParseResult) Command {
	return &SessionCommand{
		Command: *inv.Args["command"],
	}
}

// Execute a contract call
func (c *SessionCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot manage session", cliutil.ErrWalletClosed)
	}

	result := NewExecutionResult()

	switch c.Command {
	case "begin":
		err := ee.Session.BeginSession()
		if err != nil {
			return nil, fmt.Errorf("cannot begin transaction session, %w", err)
		}
		result.AddMessage("Began transaction session")
	case "submit":
		if !ee.IsWalletOpen() {
			return nil, fmt.Errorf("%w: cannot submit session", cliutil.ErrWalletClosed)
		}

		var offline bool = false

		if !ee.IsOnline() {
			if ee.IsNonceAuto() {
				return nil, fmt.Errorf("%w: cannot submit offline session if nonce is auto", cliutil.ErrOffline)
			}

			if ee.IsChainIDAuto() {
				return nil, fmt.Errorf("%w: cannot submit offline session if chain id is auto", cliutil.ErrOffline)
			}

			if !ee.rcLimit.absolute {
				return nil, fmt.Errorf("%w: cannot submit offline session if resource limit is a percentage", cliutil.ErrOffline)
			}

			// Set offline flag and continue
			offline = true
		}

		reqs, err := ee.Session.GetOperations()
		if err != nil {
			return nil, fmt.Errorf("cannot submit transaction session, %w", err)
		}

		if len(reqs) > 0 {
			ops := make([]*protocol.Operation, len(reqs))
			for i := range reqs {
				ops[i] = reqs[i].Op
			}

			if offline {
				txn, err := ee.CreateSignedTransaction(ctx, ops...)
				if err != nil {
					return nil, fmt.Errorf("cannot submit transaction session, %w", err)
				}

				// Convert to json
				result.AddMessage("JSON:")
				unformatedTxnJSON, err := kjson.Marshal(txn)
				if err != nil {
					return nil, fmt.Errorf("cannot submit transaction session, %w", err)
				}
				buffer := bytes.NewBuffer(make([]byte, 0))
				err = json.Indent(buffer, unformatedTxnJSON, "", "  ")
				if err != nil {
					return nil, fmt.Errorf("cannot submit transaction session, %w", err)
				}
				txnJSON := buffer.String()
				result.AddMessage(string(txnJSON))

				// Convert to base64
				data, err := proto.Marshal(txn)
				if err != nil {
					return nil, fmt.Errorf("cannot submit transaction session, %w", err)
				}

				result.AddMessage("\nBase64:")
				result.AddMessage(base64.URLEncoding.EncodeToString(data))
			} else {
				err := ee.SubmitTransaction(ctx, result, ops...)
				if err != nil {
					return result, fmt.Errorf("error submitting transaction, %w", err)
				}
			}
		} else {
			result.AddMessage("Cancelling transaction because session has 0 operations")
		}

		err = ee.Session.EndSession()
		if err != nil {
			return nil, fmt.Errorf("cannot end transaction session, %w", err)
		}
	case "cancel":
		err := ee.Session.EndSession()
		if err != nil {
			return nil, fmt.Errorf("cannot cancel transaction session, %w", err)
		}
		result.AddMessage("Cancelled transaction session")
	case "view":
		reqs, err := ee.Session.GetOperations()
		if err != nil {
			return nil, fmt.Errorf("cannot view transaction session, %w", err)
		}

		result.AddMessage(fmt.Sprintf("Transaction Session (%v operations):", len(reqs)))
		for i, op := range reqs {
			result.AddMessage(fmt.Sprintf("%v: %s", i, op.LogMessage))
		}
	default:
		return nil, fmt.Errorf("unknown command %s, options are (begin, submit, cancel, view)", c.Command)
	}

	return result, nil
}

// ----------------------------------------------------------------------------
// Sign Command
// ----------------------------------------------------------------------------

// SignTransactionCommand is a command that signs a transaction with the open wallet
type SignTransactionCommand struct {
	Transaction string
}

// NewSignTransactionCommand signs a transacion
func NewSignTransactionCommand(inv *CommandParseResult) Command {
	return &SignTransactionCommand{
		Transaction: *inv.Args["transaction"],
	}
}

// Execute signs a transaction
func (c *SignTransactionCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsWalletOpen() {
		return nil, fmt.Errorf("%w: cannot sign transaction", cliutil.ErrWalletClosed)
	}

	trxBytes, err := base64.URLEncoding.DecodeString(c.Transaction)
	if err != nil {
		return nil, err
	}

	trx := &protocol.Transaction{}
	err = proto.Unmarshal(trxBytes, trx)
	if err != nil {
		return nil, err
	}

	err = util.SignTransaction(ee.Key.PrivateBytes(), trx)
	if err != nil {
		return nil, err
	}

	trxBytes, err = proto.Marshal(trx)
	if err != nil {
		return nil, err
	}

	jsonTrx, err := json.MarshalIndent(trx, "", "  ")
	if err != nil {
		return nil, err
	}

	encodedTrx := base64.URLEncoding.EncodeToString(trxBytes)

	result := NewExecutionResult()
	result.AddMessage(fmt.Sprintf("Signed Transaction:\nJSON:\n%v\nBase64:\n%v", string(jsonTrx), encodedTrx))

	return result, nil
}

// ----------------------------------------------------------------------------
// AccountRc Command
// ----------------------------------------------------------------------------

// AccountRcCommand is a command that retrieves a given accounts resource credits
type AccountRcCommand struct {
	Address *string
}

// NewAccountRcCommand creates a new GetAccountRcsCommand object
func NewAccountRcCommand(inv *CommandParseResult) Command {
	return &AccountRcCommand{Address: inv.Args["address"]}
}

// Execute the retrieval of a given addresses resource credits
func (c *AccountRcCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot get account rc", cliutil.ErrOffline)
	}

	var address []byte

	if c.Address == nil {
		if !ee.IsWalletOpen() {
			return nil, fmt.Errorf("%w: cannot get account rc", cliutil.ErrWalletClosed)
		}

		address = ee.Key.AddressBytes()
	} else {
		address = base58.Decode(*c.Address)
		if len(address) == 0 {
			return nil, errors.New("could not parse address")
		}
	}

	rc, err := ee.RPCClient.GetAccountRc(ctx, address)
	if err != nil {
		return nil, err
	}

	message := fmt.Sprintf("%d.%08d rc", rc/100000000, rc%100000000)

	result := NewExecutionResult()
	result.AddMessage(message)

	return result, nil
}

// ----------------------------------------------------------------------------
// AccountNonce Command
// ----------------------------------------------------------------------------

// AccountNonceCommand is a command that retrieves a given accounts nonce
type AccountNonceCommand struct {
	Address *string
}

// NewAccountNonceCommand creates a new GetAccountNonceCommand object
func NewAccountNonceCommand(inv *CommandParseResult) Command {
	return &AccountNonceCommand{Address: inv.Args["address"]}
}

// Execute the retrieval of a given addresses nonce
func (c *AccountNonceCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	if !ee.IsOnline() {
		return nil, fmt.Errorf("%w: cannot get account nonce", cliutil.ErrOffline)
	}

	var address []byte
	if c.Address == nil {
		if !ee.IsWalletOpen() {
			return nil, fmt.Errorf("%w: cannot get account nonce", cliutil.ErrWalletClosed)
		}

		address = ee.Key.AddressBytes()
	} else {
		address = base58.Decode(*c.Address)
		if len(address) == 0 {
			return nil, errors.New("could not parse address")
		}
	}

	nonce, err := ee.RPCClient.GetAccountNonce(ctx, address)
	if err != nil {
		return nil, err
	}

	message := fmt.Sprintf("%v", nonce)

	result := NewExecutionResult()
	result.AddMessage(message)

	return result, nil
}

// ----------------------------------------------------------------------------
// List
// ----------------------------------------------------------------------------

// ListCommand is a command that lists available commands
type ListCommand struct {
}

// NewListCommand creates a new list command object
func NewListCommand(inv *CommandParseResult) Command {
	return &ListCommand{}
}

// Execute lists available commands
func (c *ListCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	cmds := ee.Parser.Commands.List(true)

	result := NewExecutionResult()
	result.AddMessage(cmds...)

	return result, nil
}
