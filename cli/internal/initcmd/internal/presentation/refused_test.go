package presentation

import (
	"testing"

	"github.com/spf13/cobra"
)

// The root names init and upgrade refuse, each naming the command under project that replaced it
// (ADR-018). Whatever was typed gets the one sentence: the command parses no flags and refuses no
// arguments, and the sentence is the error, so nothing is written.
func TestRefusalsNameTheCommandsUnderProject(t *testing.T) {
	for _, tc := range []struct {
		name     string
		refusal  func() *cobra.Command
		sentence string
	}{
		{
			name:     "init",
			refusal:  func() *cobra.Command { return NewInitRefusal() },
			sentence: "codefall init is now codefall project init",
		},
		{
			name:     "upgrade",
			refusal:  func() *cobra.Command { return NewUpgradeRefusal() },
			sentence: "codefall upgrade is now codefall project upgrade",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.refusal()
			if cmd.Name() != tc.name {
				t.Errorf("Use = %q, want %q", cmd.Name(), tc.name)
			}

			// Flags and arguments are neither parsed nor refused: the answer is the same sentence
			// with them as without.
			for _, args := range [][]string{nil, {"--harness", "claude"}, {"a", "directory"}} {
				_, err := runCommand(t, cmd, args...)
				if err == nil {
					t.Fatalf("run %v: want an error, got nil", args)
				}

				if err.Error() != tc.sentence {
					t.Errorf("run %v: error = %q, want %q", args, err.Error(), tc.sentence)
				}
			}
		})
	}
}
