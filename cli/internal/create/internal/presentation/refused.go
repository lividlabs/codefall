package presentation

import (
	"errors"

	"github.com/spf13/cobra"
)

// The root kept create's name when the command moved under codefall project, as a command that
// refuses: a person or a script that runs it learns the new name from the error rather than from a
// manual (ADR-018). The refusal is hidden from the root's help, so the group is what the help lists.

// NewCreateRefusal builds the root `codefall create`, which refuses and names the command under
// project that replaced it. The sentence is the whole answer, whatever was typed: flags and
// arguments are neither parsed nor refused, and nothing is written — the sentence is the error, and
// Fang renders it on stderr.
func NewCreateRefusal() *cobra.Command {
	return &cobra.Command{
		Use:                "create",
		Short:              "Refused; the command moved under codefall project",
		Long:               "the create command is now codefall project create.",
		Hidden:             true,
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(*cobra.Command, []string) error {
			return errors.New("the create command is now codefall project create")
		},
	}
}
