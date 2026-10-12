package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
