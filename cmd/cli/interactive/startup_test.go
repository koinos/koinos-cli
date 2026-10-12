package interactive

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"golang.org/x/term"
)

func TestStartupReviewRequiresAnAffirmativeAnswer(t *testing.T) {
	for _, test := range []struct {
		input              string
		approve, cancelled bool
	}{
		{"y\r", true, false}, {"yes\r", true, false}, {"YES\r", true, false},
		{"\r", false, false}, {"n\r", false, false}, {"maybe\r", false, false},
		{"", false, true}, {"\x03", false, true},
	} {
		t.Run(test.input, func(t *testing.T) {
			var output bytes.Buffer
			terminal := term.NewTerminal(struct {
				io.Reader
				io.Writer
			}{strings.NewReader(test.input), &output}, "Load this startup file? [y/N]: ")
			approved, err := reviewStartupFile(terminal, "/example/.koinosrc", []byte("register_token extra address\n"))
			if approved != test.approve || (err != nil) != test.cancelled {
				t.Fatalf("wrong decision: approved=%v err=%v", approved, err)
			}
			text := output.String()
			contents := strings.Index(text, "register_token extra address")
			prompt := strings.Index(text, "Load this startup file?")
			if !strings.Contains(text, "/example/.koinosrc") || contents < 0 || prompt < contents {
				t.Fatalf("file was not shown before the question: %q", text)
			}
		})
	}
}

func TestStartupPreviewEscapesTerminalControlsAndInvisibleText(t *testing.T) {
	var output bytes.Buffer
	terminal := term.NewTerminal(struct {
		io.Reader
		io.Writer
	}{strings.NewReader("n\r"), &output}, "Load? [y/N]: ")
	_, err := reviewStartupFile(terminal, "/example/\x1b[2J.koinosrc", []byte("list\r\x1b[2J\u202e\xff\n"))
	if err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, hidden := range []string{"\x1b[2J", "\u202e", "\xff"} {
		if strings.Contains(text, hidden) {
			t.Fatalf("preview emitted hidden control %q", hidden)
		}
	}
	for _, escaped := range []string{`\x1b[2J`, `\r`, `\u202e`, `\xff`} {
		if !strings.Contains(text, escaped) {
			t.Fatalf("preview concealed escaped content %q", escaped)
		}
	}
}
