package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// These exercise the composition root end to end — a real injector, real providers, the real command
// tree — but only through paths that stop at help or at argument parsing. Nothing here executes bd
// or gh.

func TestRunPrintsSubcommandHelp(t *testing.T) {
	var out bytes.Buffer

	if err := run([]string{"doctor", "--help"}, &out, &out); err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, want := range []string{"USAGE", "codefall doctor"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output does not contain %q:\n%s", want, out.String())
		}
	}
}

func TestRunWithoutArgumentsPrintsRootHelp(t *testing.T) {
	var out bytes.Buffer

	if err := run([]string{}, &out, &out); err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, want := range []string{"COMMANDS", "config", "doctor", "project", "update"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output does not contain %q:\n%s", want, out.String())
		}
	}
}

// The group prints its help when it is run alone, and the help lists the five commands it holds:
// the three that moved under it and the second spellings of config and doctor (ADR-018).
func TestProjectAlonePrintsHelpNamingItsCommands(t *testing.T) {
	var out bytes.Buffer

	if err := run([]string{"project"}, &out, &out); err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, want := range []string{"COMMANDS", "config", "create", "doctor", "init", "upgrade"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output does not contain %q:\n%s", want, out.String())
		}
	}
}

// Config and doctor stay at the root and take the prefix as a second spelling, and each mount is a
// command built for it alone (ADR-018): a *cobra.Command holds the parent it was mounted under and
// names the path down to it in its usage line, so the instance on the root cannot serve both
// places. This holds the tree to that: the prefixed commands exist beside the root's, they name the
// prefixed path, and they are not the root's instances. The prefixed config holds the subcommands
// the root's does, so a script that reaches for one under the prefix finds it.
func TestTheProjectGroupHoldsSecondMountsOfConfigAndDoctor(t *testing.T) {
	_, root := build()

	project := childNamed(root, "project")
	if project == nil {
		t.Fatal("the root does not hold the project group")
	}

	for _, name := range []string{"config", "doctor"} {
		atRoot, prefixed := childNamed(root, name), childNamed(project, name)
		if atRoot == nil {
			t.Fatalf("the root no longer holds %q", name)
		}
		if prefixed == nil {
			t.Fatalf("the project group does not hold %q", name)
		}
		if prefixed == atRoot {
			t.Errorf(
				"%s under project is the instance mounted on the root; each mount needs a command built for it alone",
				name)
		}
		if got, want := prefixed.CommandPath(), "codefall project "+name; got != want {
			t.Errorf("%s under project names its path %q, want %q", name, got, want)
		}
	}

	rootSubcommands := subcommandNames(childNamed(root, "config"))
	prefixedSubcommands := subcommandNames(childNamed(project, "config"))
	if !slices.Equal(rootSubcommands, prefixedSubcommands) {
		t.Errorf("config under project holds %v, want the root config's subcommands %v",
			prefixedSubcommands, rootSubcommands)
	}
}

// subcommandNames lists a command's visible subcommands in the order Cobra holds them, which is by
// name.
func subcommandNames(cmd *cobra.Command) []string {
	var names []string

	for _, child := range cmd.Commands() {
		if !child.Hidden {
			names = append(names, child.Name())
		}
	}

	return names
}

// The prefixed spellings print help that names the prefixed path, and config's subcommands work
// under the prefix with prefixed paths of their own (ADR-018).
func TestRunPrintsHelpForTheSecondSpellings(t *testing.T) {
	for _, args := range [][]string{
		{"project", "config", "--help"},
		{"project", "doctor", "--help"},
		{"project", "config", "show", "--help"},
		{"project", "config", "agents", "--help"},
		{"project", "config", "harness", "--help"},
		{"project", "config", "review", "--help"},
		{"project", "config", "persona", "--help"},
	} {
		var out bytes.Buffer

		if err := run(args, &out, &out); err != nil {
			t.Fatalf("run %v: %v", args, err)
		}

		// The usage line is the path the command was run under, without the --help flag.
		want := strings.Join(args[:len(args)-1], " ")
		if !strings.Contains(out.String(), want) {
			t.Errorf("run %v: output does not contain %q:\n%s", args, want, out.String())
		}
	}
}

// The names init, upgrade, and create had at the root refuse with one sentence naming the command
// under project, whatever arguments and flags they were given, and write nothing (ADR-018). The
// sentence reaches stderr through Fang's error handler, and the non-zero exit is the error run
// returns.
func TestRunRefusesTheOldRootNames(t *testing.T) {
	for _, name := range []string{"init", "upgrade", "create"} {
		for _, args := range [][]string{
			{name},
			{name, "--harness", "claude"},
			{name, "a-directory", "--remote", "git@github.com:o/app.git"},
		} {
			var out, errOut bytes.Buffer

			err := run(args, &out, &errOut)
			if err == nil {
				t.Fatalf("run %v: want an error, got nil", args)
			}

			if out.Len() != 0 {
				t.Errorf("run %v: wrote %q to stdout, want nothing", args, out.String())
			}

			// Fang's ErrorText style capitalises the first word, so the sentence is matched without
			// regard to case.
			want := "the " + name + " command is now codefall project " + name
			if !strings.Contains(strings.ToLower(errOut.String()), want) {
				t.Errorf("run %v: error does not say %q:\n%s", args, want, errOut.String())
			}
		}
	}
}

// A refused name is a command the root mounts hidden, so the help the root prints lists the group
// and not the names the group replaced.
func TestTheRootHelpHoldsNoRefusedName(t *testing.T) {
	_, root := build()

	for _, name := range []string{"init", "upgrade", "create"} {
		for _, cmd := range root.Commands() {
			if cmd.Name() == name && !cmd.Hidden {
				t.Errorf("the root lists %q in its help; the command under project replaced it", name)
			}
		}
	}

	var out bytes.Buffer

	root.SetArgs([]string{"--help"})
	root.SetOut(&out)
	root.SetErr(&out)
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	for _, name := range []string{"init", "upgrade", "create"} {
		if strings.Contains(out.String(), name) {
			t.Errorf("the root's help names %q:\n%s", name, out.String())
		}
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer

	if err := run([]string{"nope"}, &out, &out); err == nil {
		t.Fatalf("run: want an error, got nil\n%s", out.String())
	}

	// The chip and the message share a line: Fang's default puts them on two, indented, which reads
	// as a broken layout under doctor's report. The message is matched without regard to case
	// because Fang's ErrorText style capitalises its first word.
	var found bool

	for _, line := range strings.Split(out.String(), "\n") {
		if strings.Contains(line, "ERROR") && strings.Contains(strings.ToLower(line), "unknown command") {
			found = true

			break
		}
	}

	if !found {
		t.Errorf("no single line holds both %q and %q:\n%s", "ERROR", "unknown command", out.String())
	}
}
