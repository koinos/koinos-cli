package main

import (
	"context"
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
	rpcOption     = "rpc"
	executeOption = "execute"
)

// Default options
const (
	rpcDefault     = "http://localhost:8080"
	executeDefault = ""
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
	executeCmd := flag.StringP(executeOption, "x", executeDefault, "Command to execute")

	flag.Parse()

	// Setup command execution environment
	client := jsonrpc.NewClient(*rpcAddress)
	contractID, err := internal.ContractStringToID(KoinContractID)
	if err != nil {
		panic("Invalid contract ID")
	}

	cmdEnv := internal.ExecutionEnvironment{RPCClient: client, KoinContractID: contractID, KoinBalanceOfEntry: BalanceOfEntryPoint}

	// Construct the command parser
	commands := internal.BuildCommands()
	parser := internal.NewCommandParser(commands)

	if *executeCmd != "" {
		invs, _ := parser.Parse(*executeCmd)

		// Execute commands
		for _, inv := range invs {
			cmd := inv.Instantiate()
			result, _ := cmd.Execute(context.Background(), &cmdEnv)
			fmt.Println(result.Message)
		}
	}
	/*address := types.AccountType("1Krs7v1rtpgRyfwEZncuKMQQnY5JhqXVSx")
	bcmd := internal.BalanceCommand{Address: &address}
	res, err := bcmd.Execute(context.Background(), &cmdEnv)
	if err != nil {
		panic(err)
	}

	fmt.Println(res.Message)*/
}
