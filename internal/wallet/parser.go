package wallet

import (
	"errors"
	"fmt"
	"regexp"
)

// ParseResult is the result of parsing a command string
type ParseResult struct {
	CommandName string
	Args        map[string]string // This could be a slice of strings potentially
	Decl        *CommandDeclaration
	CurrentArg  int
}

// NewParseResult creates a new parse result object
func NewParseResult(name string) *ParseResult {
	inv := &ParseResult{
		CommandName: name,
		Args:        make(map[string]string),
		CurrentArg:  -1,
	}

	return inv
}

// Instantiate creates a new command object from the invocation object
func (inv *ParseResult) Instantiate() CLICommand {
	return inv.Decl.Instantiation(inv)
}

// CommandParser is a parser for commands
type CommandParser struct {
	Commands     []*CommandDeclaration
	name2command map[string]*CommandDeclaration

	// Parser token recognizer regexps
	commandNameRE *regexp.Regexp
	skipRE        *regexp.Regexp
	terminatorRE  *regexp.Regexp
	addressRE     *regexp.Regexp
}

// NewCommandParser creates a new command parser
func NewCommandParser(commands []*CommandDeclaration) *CommandParser {
	parser := &CommandParser{
		Commands:     commands,
		name2command: make(map[string]*CommandDeclaration),
	}

	for _, command := range commands {
		parser.name2command[command.Name] = command
	}

	parser.commandNameRE = regexp.MustCompile(`^[a-zA-Z0-9_]+`)
	parser.skipRE = regexp.MustCompile(`^\s*`)
	parser.terminatorRE = regexp.MustCompile(`^(;|$)`)
	parser.addressRE = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]+`)

	return parser
}

// Parse parses a string of command(s)
func (p *CommandParser) Parse(commands string) ([]*ParseResult, error) {
	// Sanitize input string and make byte buffer
	input := []byte(commands)
	var invs []*ParseResult = make([]*ParseResult, 0)

	// Loop until we've consumed all input
	for len(input) > 0 {
		var err error
		var inv *ParseResult

		input, _ = p.parseSkip(input, inv, false)
		inv, input, err = p.parseNextCommand(input)
		if inv != nil {
			invs = append(invs, inv)
		}
		if err != nil && !errors.Is(err, ErrEmptyCommandName) {
			return invs, err
		}
	}

	return invs, nil
}

func (p *CommandParser) parseNextCommand(input []byte) (*ParseResult, []byte, error) {
	// Parse the command name
	name, err := p.parseCommandName(input)
	if err != nil {
		return nil, nil, err
	}
	// Advance the input buffer
	input = input[len(name):]

	// Create the invocation object
	inv := NewParseResult(string(name))
	if decl, ok := p.name2command[string(name)]; ok {
		inv.Decl = decl
	} else {
		p.parseSkip(input, inv, true)
		return inv, nil, fmt.Errorf("%w", ErrUnknownCommand)
	}

	// Skip whitespace
	input, _ = p.parseSkip(input, inv, true)

	input, err = p.parseArgs(input, inv)
	if err != nil {
		return inv, input, err
	}

	return inv, input, nil
}

// Returns the matched command name
func (p *CommandParser) parseCommandName(input []byte) ([]byte, error) {
	m := p.commandNameRE.Find(input)
	if m == nil {
		return nil, fmt.Errorf("%w", ErrEmptyCommandName)
	}

	return m, nil
}

// Parse a command's arguments. Returns unconsumed input
func (p *CommandParser) parseArgs(input []byte, inv *ParseResult) ([]byte, error) {
	// Loop through expected arguments
	for _, arg := range inv.Decl.Args {
		// Skip whitespace
		var t bool
		input, t = p.parseSkip(input, inv, true)
		if t {
			return input, nil
		}

		var match []byte
		var err error

		// Match the argument based on type
		switch arg.ArgType {
		case Address:
			match, err = p.parseAddress(input)
		}
		input = input[len(match):] // Consume the match

		// Check for error during match
		if err != nil {
			return input, err
		}

		// Store the argument value in the invocation
		inv.Args[arg.Name] = string(match)
	}

	return input, nil
}

// Parse an address. Returns Matched address and error
func (p *CommandParser) parseAddress(input []byte) ([]byte, error) {
	// Parse address
	m := p.addressRE.Find(input)
	if m == nil {
		return nil, fmt.Errorf("%w", ErrEmptyParam)
	}

	return m, nil
}

// Returns the rest of the string, and a bool that is true if it encountered a terminator
func (p *CommandParser) parseSkip(input []byte, inv *ParseResult, incArgs bool) ([]byte, bool) {
	term := false
	skipped := false

	m := p.skipRE.Find(input)
	if len(m) > 0 {
		skipped = true
		input = input[len(m):]
	}

	if p.terminatorRE.Match(input) {
		input = input[len(p.terminatorRE.Find(input)):]
		term = true
	}

	m = p.skipRE.Find(input)
	if len(m) > 0 {
		skipped = true
		input = input[len(m):]
	}

	if skipped && incArgs {
		inv.CurrentArg++
	}

	return input, term
}
