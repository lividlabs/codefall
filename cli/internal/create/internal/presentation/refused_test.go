package presentation

import (
	"bytes"
	"testing"
)

// The root name create refuses, naming the command under project that replaced it (ADR-018).
// Whatever was typed gets the one sentence: the command parses no flags and refuses no arguments,
// and the sentence is the error, so nothing is written.
func TestCreateRefusalNamesTheCommandUnderProject(t *testing.T) {
	cmd := NewCreateRefusal()
	if cmd.Name() != "create" {
		t.Errorf("Use = %q, want %q", cmd.Name(), "create")
	}

	// Flags and arguments are neither parsed nor refused: the answer is the same sentence with
	// them as without.
	for _, args := range [][]string{nil, {"--remote", "git@github.com:o/app.git"}, {"a", "directory"}} {
		var out bytes.Buffer

		cmd.SetArgs(args)
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		// Fang sets both on the root in production, because it renders errors and usage itself.
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		err := cmd.Execute()
		if err == nil {
			t.Fatalf("run %v: want an error, got nil", args)
		}

		if want := "codefall create is now codefall project create"; err.Error() != want {
			t.Errorf("run %v: error = %q, want %q", args, err.Error(), want)
		}

		if out.Len() != 0 {
			t.Errorf("run %v: wrote %q, want nothing", args, out.String())
		}
	}
}
