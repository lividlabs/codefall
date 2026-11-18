package main

import (
	"bytes"
	"strings"
	"testing"
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

// The group prints its help when it is run alone, and the help lists the three commands it holds
// (ADR-018).
func TestProjectAlonePrintsHelpNamingItsCommands(t *testing.T) {
	var out bytes.Buffer

	if err := run([]string{"project"}, &out, &out); err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, want := range []string{"COMMANDS", "create", "init", "upgrade"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output does not contain %q:\n%s", want, out.String())
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
			want := "codefall " + name + " is now codefall project " + name
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
