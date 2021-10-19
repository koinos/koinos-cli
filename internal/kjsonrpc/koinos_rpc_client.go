package kjsonrpc

import (
	"crypto/sha256"

	"github.com/koinos/koinos-cli-wallet/internal/util"
	"github.com/koinos/koinos-proto-golang/koinos/canonical"
	"github.com/koinos/koinos-proto-golang/koinos/contracts/token"
	"github.com/koinos/koinos-proto-golang/koinos/protocol"
	"github.com/koinos/koinos-proto-golang/koinos/rpc/chain"
	"github.com/multiformats/go-multihash"
	"google.golang.org/protobuf/proto"
)

const (
	ReadContractCall      = "chain.read_contract"
	GetAccountNonceCall   = "chain.get_account_nonce"
	GetAccountRcCall      = "chain.get_account_rc"
	SubmitTransactionCall = "chain.submit_transaction"
)

// KoinosRPCClient is a wrapper around the jsonrpc client
type KoinosRPCClient struct {
	client RPCClient
}

// NewKoinosRPCClient creates a new koinos rpc client
func NewKoinosRPCClient(url string) *KoinosRPCClient {
	client := NewClient(url)
	return &KoinosRPCClient{client: client}
}

// Call wraps the rpc client call and handles some of the boilerplate
func (c *KoinosRPCClient) Call(method string, params proto.Message, returnType proto.Message) error {
	// Make the rpc call
	resp, err := c.client.Call(method, params)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}

	// Fetch the contract response
	err = resp.GetObject(returnType)
	if err != nil {
		return err
	}

	return nil
}

// GetAccountBalance gets the balance of a given account
func (c *KoinosRPCClient) GetAccountBalance(address []byte, contractID []byte, balanceOfEntry uint32) (uint64, error) {
	// Make the rpc call
	balanceOfArgs := &token.BalanceOfArguments{
		Owner: address,
	}
	argBytes, err := proto.Marshal(balanceOfArgs)
	if err != nil {
		return 0, err
	}

	cResp, err := c.ReadContract(argBytes, contractID, balanceOfEntry)
	if err != nil {
		return 0, err
	}

	balanceOfReturn := &token.BalanceOfResult{}
	err = proto.Unmarshal(cResp.Result, balanceOfReturn)
	if err != nil {
		return 0, err
	}

	return balanceOfReturn.Value, nil
}

// ReadContract reads from the given contract and returns the response
func (c *KoinosRPCClient) ReadContract(args []byte, contractID []byte, entryPoint uint32) (*chain.ReadContractResponse, error) {
	// Build the contract request
	params := chain.ReadContractRequest{ContractId: contractID, EntryPoint: entryPoint, Args: args}

	// Make the rpc call
	var cResp chain.ReadContractResponse
	err := c.Call(ReadContractCall, &params, &cResp)
	if err != nil {
		return nil, err
	}

	return &cResp, nil
}

func (c *KoinosRPCClient) WriteMessageContract(msg proto.Message, key *util.KoinosKey, contractID []byte, entryPoint uint32) (*chain.SubmitTransactionResponse, error) {
	args, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return c.WriteContract(args, key, contractID, entryPoint)
}

func (c *KoinosRPCClient) WriteContract(args []byte, key *util.KoinosKey, contractID []byte, entryPoint uint32) (*chain.SubmitTransactionResponse, error) {
	// Cache the public address
	address := key.AddressBytes()

	// Fetch the account's nonce
	nonce, err := c.GetAccountNonce(address)
	if err != nil {
		return nil, err
	}

	// Create the operation
	callContractOp := protocol.CallContractOperation{ContractId: contractID, EntryPoint: entryPoint, Args: args}
	cco := protocol.Operation_CallContract{CallContract: &callContractOp}
	op := protocol.Operation{Op: &cco}

	rcLimit, err := c.GetAccountRc(address)
	if err != nil {
		return nil, err
	}

	// Create the transaction
	active := protocol.ActiveTransactionData{Nonce: nonce, Operations: []*protocol.Operation{&op}, RcLimit: rcLimit}
	activeBytes, err := canonical.Marshal(&active)
	if err != nil {
		return nil, err
	}

	// Calculate the transaction ID
	sha256Hasher := sha256.New()
	sha256Hasher.Write(activeBytes)

	tid, err := multihash.EncodeName(sha256Hasher.Sum(nil), "sha2-256")
	if err != nil {
		return nil, err
	}
	transaction := protocol.Transaction{Active: activeBytes, Id: tid}

	// Sign the transaction
	err = util.SignTransaction(key.PrivateBytes(), &transaction)

	if err != nil {
		return nil, err
	}

	// Submit the transaction
	params := chain.SubmitTransactionRequest{}
	params.Transaction = &transaction

	// Make the rpc call
	var cResp chain.SubmitTransactionResponse
	err = c.Call(SubmitTransactionCall, &params, &cResp)
	if err != nil {
		return nil, err
	}

	return &cResp, nil
}

// GetAccountRc gets the rc of a given account
func (c *KoinosRPCClient) GetAccountRc(address []byte) (uint64, error) {
	// Build the contract request
	params := chain.GetAccountRcRequest{
		Account: address,
	}

	// Make the rpc call
	var cResp chain.GetAccountRcResponse
	err := c.Call(GetAccountRcCall, &params, &cResp)
	if err != nil {
		return 0, err
	}

	return cResp.Rc, nil
}

// GetAccountNonce gets the nonce of a given account
func (c *KoinosRPCClient) GetAccountNonce(address []byte) (uint64, error) {
	// Build the contract request
	params := chain.GetAccountNonceRequest{
		Account: address,
	}

	// Make the rpc call
	var cResp chain.GetAccountNonceResponse
	err := c.Call(GetAccountNonceCall, &params, &cResp)
	if err != nil {
		return 0, err
	}

	return cResp.Nonce, nil
}
