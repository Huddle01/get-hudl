package cli

import (
	"bytes"
	"testing"

	"github.com/Huddle01/get-hudl/internal/config"
	"github.com/Huddle01/get-hudl/internal/runtime"
)

func TestRootHelpIncludesCoreCommands(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	root := NewRootCommand(bytes.NewBuffer(nil), stdout, stderr, "test")
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute help: %v", err)
	}

	text := stdout.String()
	for _, expected := range []string{"vm", "gpu", "login", "completion"} {
		if !bytes.Contains([]byte(text), []byte(expected)) {
			t.Fatalf("expected help output to contain %q, got:\n%s", expected, text)
		}
	}
}

func TestOutputModePrefersJSONForNonTTY(t *testing.T) {
	app := &runtime.App{Config: config.Resolved{}, IsTTYOut: false}
	if got := outputMode(app); got != "json" {
		t.Fatalf("expected json output mode for non-tty, got %q", got)
	}
}
