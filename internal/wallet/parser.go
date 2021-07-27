package wallet

import (
	"fmt"
	"regexp"
)

// TerminationStatus is an enum
type TerminationStatus int

// Types of termination
const (
	None TerminationStatus = iota
	Input
	Command
)

// Characters used in parsing
const (
	CommandTerminator = ';'
)

// ParseResult is the result of parsing a command string
type ParseResult struct {
	CommandName string
	Args        map[string]string // This could be a slice of strings potentially
	Decl        *CommandDeclaration
	CurrentArg  int
	Termination TerminationStatus
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
	commandNameRE  *regexp.Regexp
	skipRE         *regexp.Regexp
	terminatorRE   *regexp.Regexp
	addressRE      *regexp.Regexp
	simpleStringRE *regexp.Regexp
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
	parser.simpleStringRE = regexp.MustCompile(`^[^\s"\';]+`)

	return parser
}

// Parse parses a string of command(s)
func (p *CommandParser) Parse(commands string) ([]*ParseResult, error) {
	// Sanitize input string and make byte buffer
	input := []byte(commands)
	var invs []*ParseResult = make([]*ParseResult, 0)

	input, _ = p.parseSkip(input, nil, false)

	// Loop until we've consumed all input
	for len(input) > 0 {
		var err error
		var inv *ParseResult

		inv, input, err = p.parseNextCommand(input)
		if inv != nil {
			invs = append(invs, inv)
		}
		if err != nil {
			return invs, err
		}

		// If latest command has no terminator or is the last command, halt parsing
		if inv.Termination == None || inv.Termination == Input {
			break
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

	input, err = p.parseArgs(input, inv)
	if err != nil {
		return inv, input, err
	}

	// Skip space and check termination
	var t TerminationStatus
	input, t = p.parseSkip(input, inv, false)
	inv.Termination = t

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
		var t TerminationStatus
		input, t = p.parseSkip(input, inv, true)
		if t != None {
			return input, fmt.Errorf("%w: %s", ErrMissingParam, arg.Name)
		}

		var match []byte
		var err error
		var l int

		// Match the argument based on type
		switch arg.ArgType {
		case Address:
			match, l, err = p.parseAddress(input)
		case String:
			match, l, err = p.parseString(input)
		}
		input = input[l:] // Consume the match

		// Check for error during match
		if err != nil {
			return input, err
		}

		// Store the argument value in the invocation
		inv.Args[arg.Name] = string(match)
	}

	return input, nil
}

// Parse an address. Returns matched address consumed length, and error
func (p *CommandParser) parseAddress(input []byte) ([]byte, int, error) {
	// Parse address
	m := p.addressRE.Find(input)
	if m == nil {
		return nil, 0, fmt.Errorf("%w", ErrMissingParam)
	}

	return m, len(m), nil
}

// Parse a string, return matched string and error
func (p *CommandParser) parseString(input []byte) ([]byte, int, error) {
	// Parse string
	if len(input) == 0 {
		return nil, 0, fmt.Errorf("%w", ErrMissingParam)
	}

	if input[0] == '"' || input[0] == '\'' {
		return p.parseQuotedString(input)
	}

	return p.parseSimpleString(input)
}

func (p *CommandParser) parseQuotedString(input []byte) ([]byte, int, error) {
	// Record the quote type
	quote := input[0]

	output := make([]byte, 0)
	escape := false // True if we're inside an escape sequence

	// Interate through the input until we find the closing quote
	for i, c := range input[1:] {
		if escape {
			escape = false

			// If we're in an escape sequence, append the character and continue to the next character
			if c == '\\' || c == '"' || c == '\'' {
				output = append(output, c)
				continue
			}

			// Otherwise just append the slash and carry on parsing this character
			output = append(output, '\\')
		}

		// If we're in an escape sequence, continue to the next character
		if c == '\\' {
			escape = true
			continue
		}

		// If end quote, return the string
		if c == quote {
			// Return the matched string
			return output, i + 2, nil
		}

		output = append(output, c)
	}

	return nil, 0, fmt.Errorf("%w: missing closing quote", ErrInvalidString)
}

func (p *CommandParser) parseSimpleString(input []byte) ([]byte, int, error) {
	m := p.simpleStringRE.Find(input)
	if m == nil {
		return nil, 0, fmt.Errorf("%w", ErrMissingParam)
	}

	return m, len(m), nil
}

// Returns the rest of the string, a bool that is true if it encountered a terminator, and a bool that is true if that terminator was a command terminator
func (p *CommandParser) parseSkip(input []byte, inv *ParseResult, incArgs bool) ([]byte, TerminationStatus) {
	term := None
	skipped := false

	m := p.skipRE.Find(input)
	if len(m) > 0 {
		skipped = true
		input = input[len(m):]
	}

	if p.terminatorRE.Match(input) {
		t := p.terminatorRE.Find(input)
		input = input[len(t):]
		if len(t) > 0 && t[0] == CommandTerminator {
			term = Command
		} else {
			term = Input
		}
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
