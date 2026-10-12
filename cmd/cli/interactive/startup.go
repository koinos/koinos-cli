package interactive

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// StartupFilePrompt is available only when input and the preview output are
// terminals. Redirected input cannot approve an automatically discovered file.
func StartupFilePrompt() func(string, []byte) (bool, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) || !term.IsTerminal(int(os.Stderr.Fd())) {
		return nil
	}
	return func(file string, contents []byte) (bool, error) {
		state, err := term.MakeRaw(fd)
		if err != nil {
			return false, err
		}
		defer term.Restore(fd, state)
		terminal := term.NewTerminal(struct {
			io.Reader
			io.Writer
		}{os.Stdin, os.Stderr}, "Load this startup file? [y/N]: ")
		return reviewStartupFile(terminal, file, contents)
	}
}

func reviewStartupFile(terminal *term.Terminal, file string, contents []byte) (bool, error) {
	var preview strings.Builder
	fmt.Fprintf(&preview, "\nStartup file: %s\n", strconv.QuoteToASCII(file))
	preview.WriteString("This file can execute CLI commands. Review it before loading.\n")
	preview.WriteString("Escaped characters are shown literally (for example, \\t and \\x1b).\n\n")
	for number, line := range strings.Split(string(contents), "\n") {
		// Quote instead of emitting terminal controls, invisible characters, or
		// invalid bytes that could conceal or overwrite the displayed commands.
		escaped := strconv.QuoteToASCII(line)
		fmt.Fprintf(&preview, "%4d | %s\n", number+1, escaped[1:len(escaped)-1])
	}
	preview.WriteByte('\n')
	if _, err := io.WriteString(terminal, preview.String()); err != nil {
		return false, err
	}
	answer, err := terminal.ReadLine()
	if err != nil {
		fmt.Fprint(terminal, "\n")
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}
