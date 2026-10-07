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

	for _, want := range []string{"COMMANDS", "config", "create", "doctor", "init", "update"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output does not contain %q:\n%s", want, out.String())
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
