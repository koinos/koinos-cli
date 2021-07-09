package internal

import (
	"regexp"
	"strings"
)

type CommandInvocation struct {
	CommandName string
	Args        map[string]string
	Decl        *CommandDeclaration
}

func NewCommandInvocation(name string) *CommandInvocation {
	inv := &CommandInvocation{
		CommandName: name,
		Args:        make(map[string]string),
	}

	return inv
}

type CommandParser struct {
	commands     []*CommandDeclaration
	name2command map[string]*CommandDeclaration

	commandNameRE *regexp.Regexp
	skipRE        *regexp.Regexp
	terminatorRE  *regexp.Regexp
}

func NewCommandParser(commands []*CommandDeclaration) *CommandParser {
	parser := &CommandParser{
		commands:     commands,
		name2command: make(map[string]*CommandDeclaration),
	}

	for _, command := range commands {
		parser.name2command[command.Name] = command
	}

	parser.commandNameRE = regexp.MustCompile(`^[a-zA-Z0-9_]+`)
	parser.skipRE = regexp.MustCompile(`^\\s*`)
	parser.terminatorRE = regexp.MustCompile(`^(;|$)`)

	return parser
}

func (p *CommandParser) Parse(commands string) ([]*CommandInvocation, error) {
	// Sanitize input string and make byte buffer
	input := []byte(strings.TrimSpace(commands))
	var invs []*CommandInvocation = make([]*CommandInvocation, 0)

	// Loop until we've consumed all input
	for len(input) > 0 {
		var err error
		var inv *CommandInvocation

		input, _ = p.parseSkip(input)
		inv, input, err = p.parseNextCommand(input)
		if invs != nil {
			invs = append(invs, inv)
		}
		if err != nil {
			return invs, err
		}
	}

	return invs, nil
}

func (p *CommandParser) parseNextCommand(input []byte) (*CommandInvocation, []byte, error) {
	// Parse the command name
	name, err := p.parseCommandName(input)
	if err != nil {
		return nil, nil, err
	}
	// Advance the input buffer
	input = input[len(name):]

	// Create the invocation object
	inv := NewCommandInvocation(string(name))
	if decl, ok := p.name2command[string(name)]; ok {
		inv.Decl = decl
	} else {
		return inv, nil, ErrUnknownCommand
	}

	// Skip to next argument
	input, t := p.parseSkip(input)
	if t {
		return inv, input, nil
	}

	return inv, input, nil
}

// Returns the matched command name
func (p *CommandParser) parseCommandName(input []byte) ([]byte, error) {
	m := p.commandNameRE.Find(input)
	if m == nil {
		return nil, ErrEmptyCommandName
	}

	return m, nil
}

// Returns the rest of the string, and a bool that is true if it encountered a terminator
func (p *CommandParser) parseSkip(input []byte) ([]byte, bool) {
	m := p.skipRE.Find(input)
	input = input[len(m):]
	if p.terminatorRE.Match(input) {
		return input[len(p.terminatorRE.Find(input)):], true
	}

	return input, false
}
