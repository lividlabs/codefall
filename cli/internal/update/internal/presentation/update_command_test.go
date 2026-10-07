package presentation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/shared/version"
	"github.com/lividlabs/codefall/cli/internal/update/internal/domain"
)

const binary = "/home/someone/.local/bin/codefall"

var (
	v28 = version.Version{Minor: 28}
	v29 = version.Version{Minor: 29}
)

type fakeUpdate struct {
	installation domain.Installation
	outcome      domain.Outcome
	err          error
	ran          bool
	requested    mo.Option[string]
}

func (f *fakeUpdate) Inspect() (domain.Installation, error)           { return f.installation, nil }
func (f *fakeUpdate) Latest(context.Context) (version.Version, error) { return v29, nil }

func (f *fakeUpdate) Run(
	_ context.Context, _ domain.Installation, requested mo.Option[string],
) (domain.Outcome, error) {
	f.ran = true
	f.requested = requested

	return f.outcome, f.err
}

// run executes the command against a fake use case and returns everything it wrote. The forced
// colour variables are cleared so ANSI cannot leak into the buffer.
func run(t *testing.T, update UpdateUseCase, args ...string) (string, error) {
	t.Helper()
	t.Setenv("CLICOLOR_FORCE", "0")
	t.Setenv("TTY_FORCE", "0")

	var out bytes.Buffer

	cmd := NewUpdateCommand(update)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()

	return out.String(), err
}

func native() domain.Installation {
	return domain.Installation{Method: domain.MethodNative, Path: binary, Release: mo.Some(v28), Build: "0.28.0"}
}

func TestUpdateReportsTheNewVersion(t *testing.T) {
	update := &fakeUpdate{
		installation: native(),
		outcome:      domain.Outcome{Path: binary, From: mo.Some(v28), To: v29, Changed: true},
	}

	out, err := run(t, update)
	if err != nil {
		t.Fatal(err)
	}

	if want := "Updated codefall from 0.28.0 to 0.29.0: " + binary + "\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestUpdatePassesTheRequestedVersion(t *testing.T) {
	update := &fakeUpdate{installation: native(), outcome: domain.Outcome{Path: binary, To: v28}}

	if _, err := run(t, update, "0.28.0"); err != nil {
		t.Fatal(err)
	}

	if update.requested != mo.Some("0.28.0") {
		t.Errorf("requested = %v", update.requested)
	}
}

func TestUpdateAlreadyUpToDate(t *testing.T) {
	update := &fakeUpdate{installation: native(), outcome: domain.Outcome{Path: binary, From: mo.Some(v29), To: v29}}

	out, err := run(t, update)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "codefall 0.29.0 is already up to date") {
		t.Errorf("output = %q", out)
	}
}

func TestUpdateCannotWrite(t *testing.T) {
	update := &fakeUpdate{installation: native(), err: fmt.Errorf("%w /usr/local/bin", domain.ErrNotWritable)}

	out, err := run(t, update)
	if !errors.Is(err, domain.ErrNotWritable) {
		t.Errorf("error = %v", err)
	}

	if !strings.Contains(out, domain.InstallScriptCommand) {
		t.Errorf("output = %q, want the install script's line", out)
	}
}

func TestUpdateLeavesOtherInstallsAlone(t *testing.T) {
	for _, tc := range []struct {
		method domain.Method
		want   []string
	}{
		{domain.MethodMise, []string{
			"installed by mise",
			"  mise upgrade " + domain.MiseTool + "\n",
			"  mise upgrade --bump " + domain.MiseTool + "\n",
		}},
		{domain.MethodSource, []string{"built from source", "  " + domain.GoInstallCommand + "\n"}},
		{domain.MethodUnknown, []string{"cannot tell how " + binary, "  " + domain.InstallScriptCommand + "\n"}},
	} {
		t.Run(tc.method.String(), func(t *testing.T) {
			update := &fakeUpdate{
				installation: domain.Installation{Method: tc.method, Path: binary, Build: "0.28.0"},
			}

			out, err := run(t, update)
			if err == nil {
				t.Error("a refused update returned no error")
			}

			if update.ran {
				t.Error("the update ran")
			}

			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("output = %q, want it to contain %q", out, want)
				}
			}
		})
	}
}

func TestUpdateCheck(t *testing.T) {
	update := &fakeUpdate{installation: domain.Installation{
		Method: domain.MethodMise, Path: binary, Build: "0.28.0",
	}}

	out, err := run(t, update, "--check")
	if err != nil {
		t.Fatal(err)
	}

	want := "Installed by  mise\nPath          " + binary + "\nVersion       0.28.0\nLatest        0.29.0\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}

	if update.ran {
		t.Error("--check ran the update")
	}
}
