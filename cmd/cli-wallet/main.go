package main

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/koinos/koinos-cli-wallet/cmd/cli-wallet/interactive"
	"github.com/koinos/koinos-cli-wallet/internal/wallet"
	types "github.com/koinos/koinos-types-golang"
	flag "github.com/spf13/pflag"
)

// Commpand line parameter names
const (
	rpcOption     = "rpc"
	executeOption = "execute"
)

// Default options
const (
	rpcDefault     = ""
	executeDefault = ""
)

// Koin contract constants
const (
	KoinContractID      = "kw96mR+Hh71IWwJoT/2lJXBDl5Q="
	BalanceOfEntryPoint = types.UInt32(0x15619248)
	TransferEntryPoint  = types.UInt32(0x62efa292)
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
	contractID, err := wallet.ContractStringToID(KoinContractID)
	if err != nil {
		panic("Invalid contract ID")
	}

	// Setup client
	var client *wallet.KoinosRPCClient
	if *rpcAddress != "" {
		client = wallet.NewKoinosRPCClient(*rpcAddress)
	}

	cmdEnv := wallet.ExecutionEnvironment{RPCClient: client, KoinContractID: contractID, KoinBalanceOfEntry: BalanceOfEntryPoint, KoinTransferEntry: TransferEntryPoint}

	// Construct the command parser
	commands := wallet.BuildCommands()
	parser := wallet.NewCommandParser(commands)

	// If the user submitted commands, execute them
	if *executeCmd != "" {
		results := wallet.ParseAndInterpret(parser, &cmdEnv, *executeCmd)
		results.Print()
		// Otherwise run the interactive shell
	} else {
		// Enter interactive mode
		p := interactive.NewKoinosPrompt(parser, &cmdEnv)
		p.Run()
	}
}
