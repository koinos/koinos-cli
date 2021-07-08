package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/joho/godotenv"
	"github.com/koinos/koinos-cli-wallet/internal"
	types "github.com/koinos/koinos-types-golang"
	flag "github.com/spf13/pflag"
	"github.com/ybbus/jsonrpc/v2"
	//"github.com/c-bata/go-prompt"
)

// Commpand line parameter names
const (
	rpcOption = "rpc"
)

// Default options
const (
	rpcDefault = "http://localhost:8080"
)

const (
	KoinContractID      = "kw96mR+Hh71IWwJoT/2lJXBDl5Q="
	BalanceOfEntryPoint = types.UInt32(0x15619248)
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}

	// Setup command line options
	rpcAddress := flag.StringP(rpcOption, "r", rpcDefault, "RPC server URL")

	flag.Parse()

	// Setup command execution environment
	client := jsonrpc.NewClient(*rpcAddress)
	contractID, err := contractStringToID(KoinContractID)
	if err != nil {
		panic("Invalid contract ID")
	}

	cmdEnv := internal.ExecutionEnvironment{RPCClient: client, KoinContractID: contractID, KoinBalanceOfEntry: BalanceOfEntryPoint}

	// Execute command
	address := types.AccountType("1Krs7v1rtpgRyfwEZncuKMQQnY5JhqXVSx")
	bcmd := internal.BalanceCommand{Address: &address}
	res, err := bcmd.Execute(context.Background(), &cmdEnv)
	if err != nil {
		panic(err)
	}

	fmt.Println(res.Message)
}

func contractStringToID(s string) (*types.ContractIDType, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	cid := types.NewContractIDType()
	if err != nil {
		return cid, err
	}

	copy(cid[:], b)
	return cid, nil
}
