package wallet

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	types "github.com/koinos/koinos-types-golang"
	"github.com/minio/sio"
	"github.com/shopspring/decimal"
	"github.com/ybbus/jsonrpc/v2"
)

// SignTransaction signs the transaction with the given key
func SignTransaction(key []byte, tx *types.Transaction) error {
	privateKey, err := crypto.ToECDSA(key)

	if err != nil {
		return err
	}

	// Sign the transaction ID
	blobID := tx.ID.Serialize(types.NewVariableBlob())
	signatureBytes, err := crypto.Sign([]byte(*blobID), privateKey)

	// Attach the signature data to the transaction
	tx.SignatureData = types.VariableBlob(signatureBytes)

	if err != nil {
		return err
	}

	return nil
}

// ContractStringToID converts a base64 contract id string to a contract id object
func ContractStringToID(s string) (*types.ContractIDType, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	cid := types.NewContractIDType()
	if err != nil {
		return cid, err
	}

	copy(cid[:], b)
	return cid, nil
}

// SatoshiToDecimal converts the given UInt64 value to a decimals with the given precision
func SatoshiToDecimal(balance int64, precision int) (*decimal.Decimal, error) {
	divisor, err := decimal.NewFromString(fmt.Sprintf("1e%d", precision))
	if err != nil {
		return nil, err
	}

	v := decimal.NewFromInt(balance).Div(divisor)
	return &v, nil
}

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
func (c *KoinosRPCClient) Call(method string, params interface{}, returnType interface{}) error {
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

func walletConfig(password []byte) sio.Config {
	return sio.Config{
		MinVersion:     sio.Version20,
		MaxVersion:     sio.Version20,
		CipherSuites:   []byte{sio.AES_256_GCM, sio.CHACHA20_POLY1305},
		Key:            password,
		SequenceNumber: uint32(0),
	}
}

// CreateWalletFile creates a new wallet file on disk
func CreateWalletFile(file *os.File, passphrase string, privateKey []byte) error {
	hasher := sha256.New()
	bytesWritten, err := hasher.Write([]byte(passphrase))

	if err != nil {
		return err
	}

	if bytesWritten <= 0 {
		return ErrEmptyPassphrase
	}

	passwordHash := hasher.Sum(nil)

	if len(passwordHash) != 32 {
		return ErrUnexpectedHashLength
	}

	source := bytes.NewReader(privateKey)
	_, err = sio.Encrypt(file, source, walletConfig(passwordHash))

	return err
}

// ReadWalletFile extracts the private key from the provided wallet file
func ReadWalletFile(file *os.File, passphrase string) ([]byte, error) {
	hasher := sha256.New()
	bytesWritten, err := hasher.Write([]byte(passphrase))

	if err != nil {
		return nil, err
	}

	if bytesWritten <= 0 {
		return nil, ErrEmptyPassphrase
	}

	passwordHash := hasher.Sum(nil)

	if len(passwordHash) != 32 {
		return nil, ErrUnexpectedHashLength
	}

	var destination bytes.Buffer
	_, err = sio.Decrypt(&destination, file, walletConfig(passwordHash))

	return destination.Bytes(), err
}

// ParseAndInterpret is a helper function to parse and interpret the given command string
func ParseAndInterpret(parser *CommandParser, ee *ExecutionEnvironment, input string) *InterpretResults {
	result, err := parser.Parse(input)
	if err != nil {
		o := NewInterpretResults()
		o.AddResult(err.Error())
		return o
	}

	return InterpretParseResults(result, ee)
}
