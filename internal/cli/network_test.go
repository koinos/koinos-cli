package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/koinos/koinos-cli/internal/cliutil"
)

func TestConnectHonorsCancellationAndKeepsCurrentNode(t *testing.T) {
	ee := NewExecutionEnvironment(cliutil.NewKoinosRPCClient("https://previous.example"), NewCommandParser(NewKoinosCommandSet()))
	previous := ee.RPCClient
	endpoint := chainEndpoint(t, base64.URLEncoding.EncodeToString([]byte("chain")))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (&ConnectCommand{URL: endpoint}).Execute(ctx, ee); err == nil {
		t.Fatal("cancelled connection succeeded")
	}
	if ee.RPCClient != previous {
		t.Fatal("failed connection discarded previous node")
	}
}

func TestPromptReflectsOfflineAndPendingSession(t *testing.T) {
	ee := NewExecutionEnvironment(nil, nil)
	if got := ee.PromptStatus(); !strings.Contains(got, "offline") || !strings.Contains(got, "locked") {
		t.Fatal(got)
	}
	if err := ee.Session.BeginSession(); err != nil {
		t.Fatal(err)
	}
	if err := ee.Session.AddOperation(nil, "pending test operation"); err != nil {
		t.Fatal(err)
	}
	if got := ee.PromptStatus(); !strings.Contains(got, "1 pending") {
		t.Fatal(got)
	}
}

func TestStatusDoesNotRevealEndpointCredentials(t *testing.T) {
	ee := NewExecutionEnvironment(cliutil.NewKoinosRPCClient("https://alice:private-password@node.example/rpc?token=private-token"), nil)
	result, err := (&StatusCommand{}).Execute(context.Background(), ee)
	if err != nil {
		t.Fatal(err)
	}
	output := strings.Join(result.Message, "\n")
	if strings.Contains(output, "private-") || strings.Contains(output, "alice") {
		t.Fatal("status revealed endpoint credentials")
	}
	if !strings.Contains(output, "https://node.example/rpc") || !strings.Contains(output, "unverified") {
		t.Fatal(output)
	}
}

func chainEndpoint(t *testing.T, chainID string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Method != cliutil.GetChainIDCall {
			t.Errorf("unexpected method %s", request.Method)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"chain_id":%q}}`, request.ID, chainID)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestConnectRejectsUnreachableNode(t *testing.T) {
	ee := NewExecutionEnvironment(nil, NewCommandParser(NewKoinosCommandSet()))
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	_, err := (&ConnectCommand{URL: server.URL}).Execute(context.Background(), ee)
	if err == nil || ee.IsOnline() {
		t.Fatal("unreachable node was reported as connected")
	}
}

func TestStatusIdentifiesReportedNetwork(t *testing.T) {
	ee := NewExecutionEnvironment(nil, NewCommandParser(NewKoinosCommandSet()))
	endpoint := chainEndpoint(t, "EiAIKVvm6-V2qmsmUvPJy09vCCLbtn9lHFpwrJbcTIEWRQ==")
	result := ParseAndInterpret(ee.Parser, ee, "connect "+endpoint+"; status")
	output := strings.Join(result.Results, "\n")
	if result.Err != nil || !strings.Contains(output, "testnet") || !strings.Contains(output, "Wallet: locked") || !strings.Contains(output, endpoint) {
		t.Fatalf("missing connection identity: %s", output)
	}
}

func TestConnectPreservesNetworkBoundStateOnWrongChain(t *testing.T) {
	parser := NewCommandParser(NewKoinosCommandSet())
	ee := NewExecutionEnvironment(nil, parser)
	first := chainEndpoint(t, base64.URLEncoding.EncodeToString([]byte("first-chain")))
	second := chainEndpoint(t, base64.URLEncoding.EncodeToString([]byte("second-chain")))
	if result := ParseAndInterpret(parser, ee, "connect "+first); result.Err != nil {
		t.Fatal(result.Err)
	}
	if result := ParseAndInterpret(parser, ee, "register_token coin 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL COIN 8"); result.Err != nil {
		t.Fatal(result.Err)
	}
	previous := ee.RPCClient
	if result := ParseAndInterpret(parser, ee, "connect "+second); result.Err == nil {
		t.Fatal("switched chains with old contract registrations")
	}
	if previous != ee.RPCClient {
		t.Fatal("failed switch replaced the active node")
	}
}

func TestConnectDoesNotSwitchDuringTransactionSession(t *testing.T) {
	ee := NewExecutionEnvironment(nil, NewCommandParser(NewKoinosCommandSet()))
	if err := ee.Session.BeginSession(); err != nil {
		t.Fatal(err)
	}
	endpoint := chainEndpoint(t, base64.URLEncoding.EncodeToString([]byte("chain")))
	if _, err := (&ConnectCommand{URL: endpoint}).Execute(context.Background(), ee); err == nil {
		t.Fatal("changed connection during a transaction session")
	}
}

func TestConnectInstallsStandardTokensForVerifiedChain(t *testing.T) {
	for _, test := range []struct {
		name, chainID, koin, vhp, koinSymbol, vhpSymbol string
	}{
		{"mainnet", "EiBZK_GGVP0H_fXVAM3j6EAuz3-B-l3ejxRSewi7qIBfSA==", "19GYjDBVXU7keLbYvMLazsGQn3GTWHjHkK", "12Y5vW6gk8GceH53YfRkRre2Rrcsgw7Naq", "KOIN", "VHP"},
		{"testnet", "EiAIKVvm6-V2qmsmUvPJy09vCCLbtn9lHFpwrJbcTIEWRQ==", "1FaSvLjQJsCJKq5ybmGsMMQs8RQYyVv8ju", "17n12ktwN79sR6ia9DDgCfmw77EgpbTyBi", "tKOIN", "VHP"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ee := NewExecutionEnvironment(nil, NewCommandParser(NewKoinosCommandSet()))
			endpoint := chainEndpoint(t, test.chainID)
			if err := ee.Connect(context.Background(), endpoint); err != nil {
				t.Fatal(err)
			}
			for _, coin := range []struct{ name, address, symbol string }{{"koin", test.koin, test.koinSymbol}, {"vhp", test.vhp, test.vhpSymbol}} {
				parsed, err := ee.Parser.Parse(coin.name + ".balance_of")
				if err != nil {
					t.Fatalf("standard token command missing: %v", err)
				}
				balance := parsed.CommandResults[0].Instantiate().(*TokenBalanceCommand)
				if base58.Encode(balance.ContractID) != coin.address || balance.Precision != 8 || balance.Symbol != coin.symbol {
					t.Fatalf("%s uses wrong network address or amounts: %+v", coin.name, balance)
				}
				if result := ParseAndInterpret(ee.Parser, ee, "help "+coin.name+".transfer; help "+coin.name+".total_supply"); result.Err != nil {
					t.Fatal(result.Err)
				}
			}
			// Reconnecting and legacy configs must not duplicate help/completion entries.
			before := ee.Parser.Commands.List(false)
			if err := ee.Connect(context.Background(), endpoint); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, ee.Parser.Commands.List(false)) {
				t.Fatal("reconnecting duplicated commands")
			}
		})
	}
}

func TestUnknownChainDoesNotInstallStandardTokens(t *testing.T) {
	ee := NewExecutionEnvironment(nil, NewCommandParser(NewKoinosCommandSet()))
	if err := ee.Connect(context.Background(), chainEndpoint(t, "AQ==")); err != nil {
		t.Fatal(err)
	}
	if _, err := ee.Parser.Parse("koin.balance_of"); err == nil {
		t.Fatal("assumed a KOIN address for an unknown chain")
	}
}

func TestStandardTokenConflictLeavesConnectionAndCommandsUnchanged(t *testing.T) {
	ee := NewExecutionEnvironment(nil, NewCommandParser(NewKoinosCommandSet()))
	// The second built-in conflicts: the first must not be partially installed.
	if result := ParseAndInterpret(ee.Parser, ee, "register_token vhp 15DJN4a8SgrbGhhGksSBASiSYjGnMU8dGL CUSTOM 8"); result.Err != nil {
		t.Fatal(result.Err)
	}
	before := ee.Parser.Commands.List(false)
	endpoint := chainEndpoint(t, "EiAIKVvm6-V2qmsmUvPJy09vCCLbtn9lHFpwrJbcTIEWRQ==")
	if err := ee.Connect(context.Background(), endpoint); err == nil {
		t.Fatal("connection replaced a conflicting token registration")
	}
	if ee.IsOnline() || len(ee.connectedChainID) != 0 || !reflect.DeepEqual(before, ee.Parser.Commands.List(false)) {
		t.Fatal("failed registration partially changed the connection or commands")
	}
}

func TestStandardTokensCannotFollowAChainSwitch(t *testing.T) {
	ee := NewExecutionEnvironment(nil, NewCommandParser(NewKoinosCommandSet()))
	if err := ee.Connect(context.Background(), chainEndpoint(t, "EiBZK_GGVP0H_fXVAM3j6EAuz3-B-l3ejxRSewi7qIBfSA==")); err != nil {
		t.Fatal(err)
	}
	previous := ee.RPCClient
	endpoint := chainEndpoint(t, "EiAIKVvm6-V2qmsmUvPJy09vCCLbtn9lHFpwrJbcTIEWRQ==")
	if err := ee.Connect(context.Background(), endpoint); err == nil {
		t.Fatal("switched networks with existing standard token addresses")
	}
	if ee.RPCClient != previous || ee.NetworkLabel() != "mainnet" {
		t.Fatal("failed switch changed the active network")
	}
}
