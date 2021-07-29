package wallet

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	types "github.com/koinos/koinos-types-golang"
	"github.com/minio/sio"
	"github.com/mr-tron/base58"
	"github.com/shopspring/decimal"
	"github.com/ybbus/jsonrpc/v2"
)

// SignTransaction signs the transaction with the given key
func SignTransaction(key []byte, tx *types.Transaction) error {
	privateKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), key)

	// Sign the transaction ID
	signatureBytes, err := btcec.SignCompact(btcec.S256(), privateKey, tx.ID.Digest, true)
	if err != nil {
		return err
	}

	// Attach the signature data to the transaction
	tx.SignatureData = types.VariableBlob(signatureBytes)

	if err != nil {
		return err
	}

	return nil
}

// ContractStringToID converts a base64 contract id string to a contract id object
func ContractStringToID(s string) (*types.ContractIDType, error) {
	b, err := base64.StdEncoding.DecodeString(s[1:])
	cid := types.NewContractIDType()
	if err != nil {
		return cid, err
	}

	copy(cid[:], b)
	return cid, nil
}

// SatoshiToDecimal converts the given UInt64 value to a decimals with the given precision
func SatoshiToDecimal(balance int64, precision int) (*decimal.Decimal, error) {
	denominator, err := decimal.NewFromString(fmt.Sprintf("1e%d", precision))
	if err != nil {
		return nil, err
	}

	v := decimal.NewFromInt(balance).Div(denominator)
	return &v, nil
}

// DecimalToSatoshi converts the given decimal to a satoshi value
func DecimalToSatoshi(d *decimal.Decimal, precision int) (int64, error) {
	multiplier, err := decimal.NewFromString(fmt.Sprintf("1e%d", precision))
	if err != nil {
		return 0, err
	}

	return d.Mul(multiplier).BigInt().Int64(), nil
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

// GetAccountBalance gets the balance of a given account
func (c *KoinosRPCClient) GetAccountBalance(address *types.AccountType, contractID *types.ContractIDType, balanceOfEntry types.UInt32) (types.UInt64, error) {
	// Build the contract request
	params := types.NewReadContractRequest()
	params.ContractID = *contractID
	params.EntryPoint = balanceOfEntry
	// Serialize the args
	vb := types.NewVariableBlob()
	vb = address.Serialize(vb)
	params.Args = *vb

	// Make the rpc call
	var cResp types.ReadContractResponse
	err := c.Call(ReadContractCall, params, &cResp)
	if err != nil {
		return 0, err
	}

	_, balance, err := types.DeserializeUInt64(&cResp.Result)
	if err != nil {
		return 0, err
	}

	return *balance, nil
}

// GetAccountNonce gets the nonce of a given account
func (c *KoinosRPCClient) GetAccountNonce(address *types.AccountType) (types.UInt64, error) {
	// Build the contract request
	params := types.NewGetAccountNonceRequest()
	params.Account = *address

	// Make the rpc call
	var cResp types.GetAccountNonceResponse
	err := c.Call(GetAccountNonceCall, params, &cResp)
	if err != nil {
		return 0, err
	}

	return cResp.Nonce, nil
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

const compressMagic byte = 0x01

// DecodeWIF decodes a WIF format string into bytes
func DecodeWIF(wif string) ([]byte, error) {
	decoded, err := base58.Decode(wif)
	if err != nil {
		return nil, err
	}

	decodedLen := len(decoded)
	var compress bool

	// Length of base58 decoded WIF must be 32 bytes + an optional 1 byte
	// (0x01) if compressed, plus 1 byte for netID + 4 bytes of checksum.
	switch decodedLen {
	case 1 + btcec.PrivKeyBytesLen + 1 + 4:
		if decoded[33] != compressMagic {
			return nil, btcutil.ErrMalformedPrivateKey
		}
		compress = true
	case 1 + btcec.PrivKeyBytesLen + 4:
		compress = false
	default:
		return nil, btcutil.ErrMalformedPrivateKey
	}

	// Checksum is first four bytes of double SHA256 of the identifier byte
	// and privKey.  Verify this matches the final 4 bytes of the decoded
	// private key.
	var tosum []byte
	if compress {
		tosum = decoded[:1+btcec.PrivKeyBytesLen+1]
	} else {
		tosum = decoded[:1+btcec.PrivKeyBytesLen]
	}
	cksum := chainhash.DoubleHashB(tosum)[:4]
	if !bytes.Equal(cksum, decoded[decodedLen-4:]) {
		return nil, btcutil.ErrChecksumMismatch
	}

	//netID := decoded[0]
	privKeyBytes := decoded[1 : 1+btcec.PrivKeyBytesLen]

	return privKeyBytes, nil
}

// EncodeWIF encodes a private key into a WIF format string
func EncodeWIF(privKey []byte, compress bool, netID byte) string {
	// Precalculate size.  Maximum number of bytes before base58 encoding
	// is one byte for the network, 32 bytes of private key, possibly one
	// extra byte if the pubkey is to be compressed, and finally four
	// bytes of checksum.
	encodeLen := 1 + 32 + 4
	if compress {
		encodeLen++
	}

	a := make([]byte, 0, encodeLen)
	a = append(a, netID)
	// Pad and append bytes manually, instead of using Serialize, to
	// avoid another call to make.
	a = paddedAppend(btcec.PrivKeyBytesLen, a, privKey)
	if compress {
		a = append(a, compressMagic)
	}
	cksum := chainhash.DoubleHashB(a)[:4]
	a = append(a, cksum...)
	return base58.Encode(a)
}

// paddedAppend appends the src byte slice to dst, returning the new slice.
// If the length of the source is smaller than the passed size, leading zero
// bytes are appended to the dst slice before appending src.
func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}
