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

// CommandArgType is an enum that defines the types of arguments a command can take
type CommandArgType int

// Types of arguments
const (
	Address CommandArgType = iota
	String
	Amount
	CmdName
)

// Characters used in parsing
const (
	CommandTerminator = ';'
)

// CommandParseResult is the result of parsing a single command string
type CommandParseResult struct {
	CommandName string
	Args        map[string]string // This could be a slice of strings potentially
	Decl        *CommandDeclaration
	CurrentArg  int
	Termination TerminationStatus
}

// NewCommandParseResult creates a new parse result object
func NewCommandParseResult(name string) *CommandParseResult {
	inv := &CommandParseResult{
		CommandName: name,
		Args:        make(map[string]string),
		CurrentArg:  -1,
	}

	return inv
}

// Instantiate creates a new command object from the invocation object
func (inv *CommandParseResult) Instantiate() CLICommand {
	return inv.Decl.Instantiation(inv)
}

// ParseResults represents the result of parsing a string of commands
type ParseResults struct {
	CommandResults []*CommandParseResult
}

// NewParseResults creates a new parse results object
func NewParseResults() *ParseResults {
	return &ParseResults{CommandResults: make([]*CommandParseResult, 0)}
}

// AddResult adds a new result to the parse results
func (pr *ParseResults) AddResult(result *CommandParseResult) {
	pr.CommandResults = append(pr.CommandResults, result)
}

// Len is the number of command parse results
func (pr *ParseResults) Len() int {
	return len(pr.CommandResults)
}

// CommandParser is a parser for commands
type CommandParser struct {
	Commands *CommandSet

	// Parser token recognizer regexps
	commandNameRE  *regexp.Regexp
	skipRE         *regexp.Regexp
	terminatorRE   *regexp.Regexp
	addressRE      *regexp.Regexp
	simpleStringRE *regexp.Regexp
	amountRE       *regexp.Regexp
}

// NewCommandParser creates a new command parser
func NewCommandParser(commands *CommandSet) *CommandParser {
	parser := &CommandParser{
		Commands: commands,
	}

	parser.commandNameRE = regexp.MustCompile(`^[a-zA-Z0-9_]+`)
	parser.skipRE = regexp.MustCompile(`^\s*`)
	parser.terminatorRE = regexp.MustCompile(`^(;|$)`)
	parser.addressRE = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]+`)
	parser.simpleStringRE = regexp.MustCompile(`^[^\s"\';]+`)
	parser.amountRE = regexp.MustCompile(`^((\d+(\.\d*)?)|(\.\d+))`)

	return parser
}

// Parse parses a string of command(s)
func (p *CommandParser) Parse(commands string) (*ParseResults, error) {
	// Sanitize input string and make byte buffer
	input := []byte(commands)
	invs := NewParseResults()

	input, _ = p.parseSkip(input, nil, false)

	// Loop until we've consumed all input
	for len(input) > 0 {
		var err error
		var inv *CommandParseResult

		inv, input, err = p.parseNextCommand(input)
		if inv != nil {
			invs.AddResult(inv)
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

func (p *CommandParser) parseNextCommand(input []byte) (*CommandParseResult, []byte, error) {
	// Parse the command name
	name, err := p.parseCommandName(input)
	if err != nil {
		return nil, nil, err
	}
	// Advance the input buffer
	input = input[len(name):]

	// Create the invocation object
	inv := NewCommandParseResult(string(name))
	if decl, ok := p.Commands.Name2Command[string(name)]; ok {
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
func (p *CommandParser) parseArgs(input []byte, inv *CommandParseResult) ([]byte, error) {
	// Loop through expected arguments
	for _, arg := range inv.Decl.Args {
		// Skip whitespace
		var t TerminationStatus
		input, t = p.parseSkip(input, inv, true)
		if t != None {
			if arg.Optional {
				inv.Args[arg.Name] = ""
				return input, nil
			} else {
				return input, fmt.Errorf("%w: %s", ErrMissingParam, arg.Name)
			}
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
		case Amount:
			match, l, err = p.parseAmount(input)
		case CmdName:
			match, l, err = p.parseString(input)
		}
		input = input[l:] // Consume the match

		// Check for error during match
		if err != nil {
			return input, fmt.Errorf("%w: %s", err, arg.Name)
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
		return nil, 0, fmt.Errorf("%w", ErrInvalidParam)
	}

	return m, len(m), nil
}

func (p *CommandParser) parseAmount(input []byte) ([]byte, int, error) {
	// Parse amount
	m := p.amountRE.Find(input)
	if m == nil {
		return nil, 0, fmt.Errorf("%w", ErrInvalidParam)
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

	return nil, 0, fmt.Errorf("%w (missing closing quote)", ErrInvalidParam)
}

func (p *CommandParser) parseSimpleString(input []byte) ([]byte, int, error) {
	m := p.simpleStringRE.Find(input)
	if m == nil {
		return nil, 0, fmt.Errorf("%w", ErrInvalidParam)
	}

	return m, len(m), nil
}

// Returns the rest of the string, a bool that is true if it encountered a terminator, and a bool that is true if that terminator was a command terminator
func (p *CommandParser) parseSkip(input []byte, inv *CommandParseResult, incArgs bool) ([]byte, TerminationStatus) {
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
