package kjsonrpc

import (
	"crypto/sha256"
	"encoding/json"

	"github.com/koinos/koinos-cli/internal/util"
	kjson "github.com/koinos/koinos-proto-golang/encoding/json"
	"github.com/koinos/koinos-proto-golang/koinos/canonical"
	"github.com/koinos/koinos-proto-golang/koinos/contract_meta_store"
	"github.com/koinos/koinos-proto-golang/koinos/contracts/token"
	"github.com/koinos/koinos-proto-golang/koinos/protocol"
	"github.com/koinos/koinos-proto-golang/koinos/rpc/chain"
	contract_meta_store_rpc "github.com/koinos/koinos-proto-golang/koinos/rpc/contract_meta_store"
	"github.com/multiformats/go-multihash"
	jsonrpc "github.com/ybbus/jsonrpc/v2"
	"google.golang.org/protobuf/proto"
)

// These are the rpc calls that the wallet uses
const (
	ReadContractCall      = "chain.read_contract"
	GetAccountNonceCall   = "chain.get_account_nonce"
	GetAccountRcCall      = "chain.get_account_rc"
	SubmitTransactionCall = "chain.submit_transaction"
	GetContractMetaCall   = "contract_meta_store.get_contract_meta"
)

// KoinosRPCClient is a wrapper around the jsonrpc client
type KoinosRPCClient struct {
	client jsonrpc.RPCClient
}

// NewKoinosRPCClient creates a new koinos rpc client
func NewKoinosRPCClient(url string) *KoinosRPCClient {
	client := jsonrpc.NewClient(url)
	return &KoinosRPCClient{client: client}
}

// Call wraps the rpc client call and handles some of the boilerplate
func (c *KoinosRPCClient) Call(method string, params proto.Message, returnType proto.Message) error {
	req, err := kjson.Marshal(params)
	if err != nil {
		return err
	}

	// Make the rpc call
	resp, err := c.client.Call(method, json.RawMessage(req))
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}

	// Fetch the contract response
	raw := json.RawMessage{}

	err = resp.GetObject(&raw)
	if err != nil {
		return err
	}

	err = kjson.Unmarshal([]byte(raw), returnType)
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

// GetContractMeta gets the metadata of a given contract
func (c *KoinosRPCClient) GetContractMeta(contractID []byte) (*contract_meta_store.ContractMetaItem, error) {
	// Build the contract request
	params := contract_meta_store_rpc.GetContractMetaRequest{
		ContractId: contractID,
	}

	// Make the rpc call
	var cResp contract_meta_store_rpc.GetContractMetaResponse
	err := c.Call(GetContractMetaCall, &params, &cResp)
	if err != nil {
		return nil, err
	}

	return cResp.Meta, nil
}

// SubmitTransaction creates and submits a transaction from a list of operations
func (c *KoinosRPCClient) SubmitTransaction(ops []*protocol.Operation, key *util.KoinosKey) ([]byte, error) {
	// Cache the public address
	address := key.AddressBytes()

	// Fetch the account's nonce
	nonce, err := c.GetAccountNonce(address)
	if err != nil {
		return nil, err
	}

	rcLimit, err := c.GetAccountRc(address)
	if err != nil {
		return nil, err
	}

	// Get operation multihashes
	opHashes := make([][]byte, len(ops))
	for i, op := range ops {
		opHashes[i], err = util.HashMessage(op)
		if err != nil {
			return nil, err
		}
	}

	// Find merkle root
	merkleRoot, err := util.CalculateMerkleRoot(opHashes)
	if err != nil {
		return nil, err
	}

	// Create the header
	header := protocol.TransactionHeader{RcLimit: rcLimit, Nonce: nonce, OperationMerkleRoot: merkleRoot}
	headerBytes, err := canonical.Marshal(&header)
	if err != nil {
		return nil, err
	}

	// Calculate the transaction ID
	sha256Hasher := sha256.New()
	sha256Hasher.Write(headerBytes)
	tid, err := multihash.EncodeName(sha256Hasher.Sum(nil), "sha2-256")
	if err != nil {
		return nil, err
	}

	// Create the transaction
	transaction := protocol.Transaction{Header: &header, Operations: ops, Id: tid}

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

	return transaction.Id, nil
}
