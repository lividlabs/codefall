package presentation

import (
	"errors"

	"github.com/spf13/cobra"
)

// The root kept initcmd's names when the commands moved under codefall project, as commands that
// refuse: a person or a script that runs one learns the new name from the error rather than from a
// manual (ADR-018). A refusal is hidden from the root's help, so the group is what the help lists.

// NewInitRefusal builds the root `codefall init`, which refuses and names the command under project
// that replaced it.
func NewInitRefusal() *cobra.Command {
	return newRefusal("init", "the init command is now codefall project init")
}

// NewUpgradeRefusal builds the root `codefall upgrade`, which refuses and names the command under
// project that replaced it.
func NewUpgradeRefusal() *cobra.Command {
	return newRefusal("upgrade", "the upgrade command is now codefall project upgrade")
}

// newRefusal builds a command that refuses with one sentence naming the command under project that
// replaced it. The sentence is the whole answer, whatever was typed: flags and arguments are
// neither parsed nor refused, and nothing is written — the sentence is the error, and Fang renders
// it on stderr.
func newRefusal(name, sentence string) *cobra.Command {
	return &cobra.Command{
		Use:                name,
		Short:              "Refused; the command moved under codefall project",
		Long:               sentence + ".",
		Hidden:             true,
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE:               func(*cobra.Command, []string) error { return errors.New(sentence) },
	}
}
