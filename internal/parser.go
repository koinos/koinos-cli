package internal

type CommandInvocation struct {
	Args map[string]string
	Decl *CommandDeclaration
}

type CommandParser struct {
	commands     []*CommandDeclaration
	name2command map[string]*CommandDeclaration
}

func NewCommandParser(commands []*CommandDeclaration) *CommandParser {
	parser := &CommandParser{
		commands:     commands,
		name2command: make(map[string]*CommandDeclaration),
	}

	for _, command := range commands {
		parser.name2command[command.Name] = command
	}

	return parser
}

func Parse(command string) {

}
