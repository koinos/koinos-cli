package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/koinos/koinos-cli/internal/cliutil"
)

// These chain identities are checked when using a named network. Endpoint
// names alone are not evidence that a node serves the intended chain.
type standardToken struct {
	name, address, symbol string
	precision             int
}

var networks = map[string]struct {
	endpoint, chainID string
	tokens            []standardToken
}{
	"mainnet": {"https://api.koinos.io/", "EiBZK_GGVP0H_fXVAM3j6EAuz3-B-l3ejxRSewi7qIBfSA==", []standardToken{
		{"koin", "15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL", "KOIN", 8},
		{"vhp", "18tWNU7E4yuQzz7hMVpceb9ixmaWLVyQsr", "VHP", 8},
	}},
	"testnet": {"https://testnet.koinosfoundation.org/jsonrpc", "EiAIKVvm6-V2qmsmUvPJy09vCCLbtn9lHFpwrJbcTIEWRQ==", []standardToken{
		{"koin", "1FaSvLjQJsCJKq5ybmGsMMQs8RQYyVv8ju", "tKOIN", 8},
		{"vhp", "17n12ktwN79sR6ia9DDgCfmw77EgpbTyBi", "VHP", 8},
	}},
}

func IsNetwork(name string) bool { _, ok := networks[name]; return ok }

// Connect verifies the target before replacing the current connection. Local
// contract registrations and pending operations must not follow a chain switch.
func (ee *ExecutionEnvironment) Connect(ctx context.Context, target string) error {
	if ee.Session.IsValid() {
		return fmt.Errorf("finish or cancel the transaction session before changing connections")
	}
	endpoint, expectedID := target, ""
	if network, ok := networks[target]; ok {
		endpoint, expectedID = network.endpoint, network.chainID
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return fmt.Errorf("use mainnet, testnet, or an http(s) node URL")
	}
	client := cliutil.NewKoinosRPCClient(endpoint)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	chainID, err := client.GetChainID(ctx)
	if err != nil {
		return fmt.Errorf("could not verify node connection; check the URL and network availability")
	}
	if len(chainID) == 0 {
		return fmt.Errorf("node returned an empty chain ID")
	}
	if expectedID != "" && base64.URLEncoding.EncodeToString(chainID) != expectedID {
		return fmt.Errorf("node chain ID does not match %s; connection unchanged", target)
	}
	changed := len(ee.connectedChainID) > 0 && !bytes.Equal(ee.connectedChainID, chainID)
	if changed && len(ee.Contracts) > 0 {
		return fmt.Errorf("cannot switch chains with registered contracts; restart with --network or --rpc for the new network")
	}
	if err := ee.registerStandardTokens(ctx, chainID); err != nil {
		return fmt.Errorf("could not register standard tokens; connection unchanged: %w", err)
	}
	if changed {
		ee.nonceMap = make(map[string]*nonceInfo)
		ee.nonceMode = AutoNonce
		ee.chainID = AutoChainID
	}
	ee.RPCClient, ee.connectedChainID = client, chainID
	return nil
}

// Standard registrations depend on the verified chain ID, including when a
// custom node URL serves a known chain. They never require metadata RPC calls.
func (ee *ExecutionEnvironment) registerStandardTokens(ctx context.Context, chainID []byte) error {
	var commands []*RegisterTokenCommand
	for _, network := range networks {
		if base64.URLEncoding.EncodeToString(chainID) != network.chainID {
			continue
		}
		for _, token := range network.tokens {
			precision := strconv.Itoa(token.precision)
			command := &RegisterTokenCommand{Name: token.name, Address: token.address, Symbol: &token.symbol, Precision: &precision}
			// Check every conflict before installing any commands or changing nodes.
			if _, err := command.alreadyRegistered(ee); err != nil {
				return err
			}
			commands = append(commands, command)
		}
	}
	for _, command := range commands {
		if _, err := command.Execute(ctx, ee); err != nil {
			return err
		}
	}
	return nil
}

// NetworkLabel describes the last verified chain, not continuous connectivity.
func (ee *ExecutionEnvironment) NetworkLabel() string {
	if !ee.IsOnline() {
		return "offline"
	}
	if len(ee.connectedChainID) == 0 {
		return "unverified"
	}
	id := base64.URLEncoding.EncodeToString(ee.connectedChainID)
	for name, network := range networks {
		if id == network.chainID {
			return name
		}
	}
	return "custom"
}

// StatusCommand reports local state without a hidden network request.
type StatusCommand struct{}

func NewStatusCommand(_ *CommandParseResult) Command { return &StatusCommand{} }

func (c *StatusCommand) Execute(ctx context.Context, ee *ExecutionEnvironment) (*ExecutionResult, error) {
	result := NewExecutionResult()
	result.AddMessage("Network: " + ee.NetworkLabel())
	if ee.IsOnline() {
		result.AddMessage("Endpoint: " + ee.RPCClient.Endpoint())
	}
	if ee.IsOnline() && len(ee.connectedChainID) > 0 {
		result.AddMessage("Chain ID (verified on connect): " + base64.URLEncoding.EncodeToString(ee.connectedChainID))
	}
	wallet := "locked"
	if ee.IsWalletOpen() {
		wallet = base58.Encode(ee.Key.AddressBytes()) + " (unlocked)"
	}
	result.AddMessage("Wallet: " + wallet)
	if ee.Session.IsValid() {
		operations, _ := ee.Session.GetOperations()
		result.AddMessage(fmt.Sprintf("Transaction session: %d pending operations", len(operations)))
	} else {
		result.AddMessage("Transaction session: none")
	}
	return result, nil
}

// PromptStatus keeps the network and signing account visible in every command.
func (ee *ExecutionEnvironment) PromptStatus() string {
	parts := []string{ee.NetworkLabel()}
	if ee.IsWalletOpen() {
		address := base58.Encode(ee.Key.AddressBytes())
		parts = append(parts, address[:6]+"..."+address[len(address)-4:]+" unlocked")
	} else {
		parts = append(parts, "locked")
	}
	if ee.Session.IsValid() {
		ops, _ := ee.Session.GetOperations()
		parts = append(parts, fmt.Sprintf("%d pending", len(ops)))
	}
	return "[" + strings.Join(parts, " | ") + "] > "
}
