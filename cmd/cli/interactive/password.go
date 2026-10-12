package interactive

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// PasswordPrompt returns no reader for redirected input. In a terminal, raw
// mode lets Ctrl-C cancel entry without leaving echo disabled after a signal.
func PasswordPrompt() func(string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil
	}
	return func(label string) (string, error) {
		state, err := term.MakeRaw(fd)
		if err != nil {
			return "", err
		}
		defer term.Restore(fd, state)
		terminal := term.NewTerminal(struct {
			io.Reader
			io.Writer
		}{os.Stdin, os.Stderr}, "")
		password, err := terminal.ReadPassword(label)
		if err != nil {
			fmt.Fprint(os.Stderr, "\r\n")
		}
		return password, err
	}
}
