package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/koinos/koinos-cli/internal/cli"
)

func executeStartupFiles(homeDir string, parser *cli.CommandParser, env *cli.ExecutionEnvironment, keepConnection bool, review func(string, []byte) (bool, error)) error {
	name := ".koinosrc"
	if keepConnection && env.NetworkLabel() == "testnet" {
		// The legacy startup file contains mainnet contract addresses.
		name += ".testnet"
	}
	var seen []os.FileInfo
	for _, file := range []string{filepath.Join(homeDir, name), name} {
		file, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		info, err := os.Stat(file)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		duplicate := false
		for _, previous := range seen {
			if os.SameFile(info, previous) {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		seen = append(seen, info)
		if review == nil {
			return fmt.Errorf("automatic startup file %q requires interactive review; use --no-rc to skip configs, or --no-rc --file %q to run this file explicitly", file, file)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("startup file %q must be a regular file", file)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		approved, err := review(file, data)
		if err != nil {
			return fmt.Errorf("startup file review cancelled: %w", err)
		}
		if !approved {
			fmt.Fprintf(os.Stderr, "Skipped startup file %q\n", file)
			continue
		}
		// Execute the reviewed bytes, even if the file changes while prompting.
		if err := executeCommandData(file, data, parser, env, keepConnection); err != nil {
			return err
		}
	}
	return nil
}

func executeCommandFile(file string, parser *cli.CommandParser, env *cli.ExecutionEnvironment, keepConnection bool) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return executeCommandData(file, data, parser, env, keepConnection)
}

func executeCommandData(file string, data []byte, parser *cli.CommandParser, env *cli.ExecutionEnvironment, keepConnection bool) error {
	for lineNumber, line := range strings.Split(string(data), "\n") {
		parsed, err := parser.Parse(line)
		var result *cli.InterpretResults
		if err != nil {
			// Preserve the CLI's normal parse error and usage diagnostics.
			result = cli.ParseAndInterpret(parser, env, line)
		} else {
			if keepConnection {
				commands := parsed.CommandResults[:0]
				for _, command := range parsed.CommandResults {
					if command.CommandName == "connect" || command.CommandName == "disconnect" {
						fmt.Fprintf(os.Stderr, "Keeping selected network; skipped %s in %s:%d\n", command.CommandName, file, lineNumber+1)
						continue
					}
					commands = append(commands, command)
				}
				parsed.CommandResults = commands
			}
			result = parsed.Interpret(env)
		}
		for _, message := range result.Results {
			fmt.Println(message)
		}
		if result.Err != nil {
			return fmt.Errorf("stopped at %s:%d", file, lineNumber+1)
		}
	}
	return nil
}
