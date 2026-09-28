package presentation

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/domain"
)

// headingLines is the report's opening chip as it reaches a non-terminal stdout: colorprofile has
// stripped the colour, and the padding the chip style adds survives as ordinary spaces.
const headingLines = " DOCTOR SUMMARY \n\n"

type fakeDiagnose struct {
	report domain.Report
	err    error
	gotDir string
}

func (f *fakeDiagnose) Run(_ context.Context, dir string) (domain.Report, error) {
	f.gotDir = dir

	return f.report, f.err
}

// run executes the command against a fake use case and returns everything it wrote.
//
// CLICOLOR_FORCE and TTY_FORCE are cleared so a developer's forced-colour environment cannot leak
// ANSI into the buffer: colorprofile honours both regardless of whether the writer is a terminal.
func run(t *testing.T, diagnose DiagnoseUseCase, args ...string) (string, error) {
	t.Helper()
	t.Setenv("CLICOLOR_FORCE", "0")
	t.Setenv("TTY_FORCE", "0")

	var out bytes.Buffer

	cmd := NewDoctorCommand(diagnose)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	// Fang sets both on the root in production, because it renders errors and usage itself.
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()

	return out.String(), err
}

func TestDoctorCommandPrintsAPassingReport(t *testing.T) {
	diagnose := &fakeDiagnose{report: domain.NewReport(
		domain.CodefallDir.Pass(),
		domain.SettingsComplete.Pass(),
		domain.BeadsInstalled.PassWithDetail("bd version 1.2.2 (Homebrew)"),
		domain.GHInstalled.PassWithDetail("gh version 2.97.0 (2026-07-31)"),
		domain.GHAuthenticated.PassWithDetail("logged in as djensen47"),
	)}

	out, err := run(t, diagnose)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// A section whose passing checks reported nothing about themselves has no parenthetical.
	want := headingLines +
		"[✓] Settings\n" +
		"[✓] Beads (bd version 1.2.2 (Homebrew))\n" +
		"[✓] GitHub CLI (gh version 2.97.0 (2026-07-31), logged in as djensen47)\n" +
		"• No issues found.\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

// The installed harnesses and the configured agents are each a passing check's detail, and a
// section's header carries its passing details, so the two lists get a header each: in one they
// read as a single list of harnesses.
func TestDoctorCommandListsHarnessesAndAgentsOnSeparateLines(t *testing.T) {
	diagnose := &fakeDiagnose{report: domain.NewReport(
		domain.HarnessNames.Pass(),
		domain.HarnessesInstalled.PassWithDetail("claude"),
		domain.SharedInstalled.PassWithDetail("5 shared files"),
		domain.HarnessesLeftOver.Pass(),
		domain.AgentsRunnable.PassWithDetail("architect (codex, gpt-5-codex), subagent (current)"),
		domain.AgentsCurrent.Pass(),
	)}

	out, err := run(t, diagnose)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := headingLines +
		"[✓] Harnesses (claude, 5 shared files)\n" +
		"[✓] Agents (architect (codex, gpt-5-codex), subagent (current))\n" +
		"• No issues found.\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

// The heading is the one line that paints a background, so it is the one whose plain-text form is
// worth asserting: colorprofile has to strip the styling on a non-terminal stdout and under
// NO_COLOR, leaving the word.
func TestDoctorCommandHeadingDegradesToPlainText(t *testing.T) {
	for _, tc := range []struct {
		name    string
		noColor bool
	}{
		{name: "a non-terminal stdout"},
		{name: "NO_COLOR", noColor: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.noColor {
				t.Setenv("NO_COLOR", "1")
			}

			out, err := run(t, &fakeDiagnose{report: domain.NewReport(domain.CodefallDir.Pass())})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}

			if strings.ContainsRune(out, 0x1b) {
				t.Errorf("output = %q, want no escape sequences", out)
			}

			first, _, _ := strings.Cut(out, "\n")
			if strings.TrimSpace(first) != heading {
				t.Errorf("first line = %q, want the plain text %q", first, heading)
			}
		})
	}
}

func TestDoctorCommandPrintsWarningsWithTheirRemedyAndSucceeds(t *testing.T) {
	diagnose := &fakeDiagnose{report: domain.NewReport(
		domain.BeadsInstalled.PassWithDetail("bd version 1.2.2 (Homebrew)"),
		domain.BeadsInitialized.Warn(
			"This repository has no Beads database (bd info exited 1: Error: no beads database found)",
			mo.Some("bd init")),
	)}

	out, err := run(t, diagnose)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := headingLines +
		"[!] Beads (bd version 1.2.2 (Homebrew))\n" +
		"╰─ ! This repository has no Beads database (bd info exited 1: Error: no beads database found)\n" +
		"   ╰─ fix: bd init\n" +
		"• Doctor found issues in 1 category.\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

func TestDoctorCommandFailsWhenAnyCheckFails(t *testing.T) {
	for _, tc := range []struct {
		name    string
		report  domain.Report
		wantErr string
		want    string
	}{
		{
			name: "one category",
			report: domain.NewReport(
				domain.CodefallDir.Pass(),
				domain.SettingsFile.Fail(".codefall/settings.json not found", mo.None[string]()),
			),
			wantErr: "doctor found issues in 1 category",
			want: headingLines +
				"[✗] Settings\n" +
				"╰─ ✗ .codefall/settings.json not found\n",
		},
		{
			name: "a failure and a warning in different categories",
			report: domain.NewReport(
				domain.CodefallDir.Pass(),
				domain.SettingsFile.Fail(".codefall/settings.json not found",
					mo.Some("create .codefall/settings.json")),
				domain.BeadsInstalled.PassWithDetail("bd version 1.2.2 (Homebrew)"),
				domain.BeadsInitialized.Warn("This repository has no Beads database", mo.Some("bd init")),
				domain.GHInstalled.PassWithDetail("gh version 2.97.0 (2026-07-31)"),
				domain.GHAuthenticated.PassWithDetail("logged in as djensen47"),
			),
			wantErr: "doctor found issues in 2 categories",
			want: headingLines +
				"[✗] Settings\n" +
				"╰─ ✗ .codefall/settings.json not found\n" +
				"   ╰─ fix: create .codefall/settings.json\n" +
				"[!] Beads (bd version 1.2.2 (Homebrew))\n" +
				"╰─ ! This repository has no Beads database\n" +
				"   ╰─ fix: bd init\n" +
				"[✓] GitHub CLI (gh version 2.97.0 (2026-07-31), logged in as djensen47)\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := run(t, &fakeDiagnose{report: tc.report})
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("Execute error = %v, want %q", err, tc.wantErr)
			}

			// The whole report is printed before the error is returned, and no summary line with it:
			// Fang renders the error as the summary.
			if out != tc.want {
				t.Errorf("output =\n%q\nwant\n%q", out, tc.want)
			}
		})
	}
}

func TestDoctorCommandWrapsAUseCaseError(t *testing.T) {
	failure := errors.New("diagnose: context canceled")

	out, err := run(t, &fakeDiagnose{err: failure})
	if !errors.Is(err, failure) {
		t.Fatalf("Execute error = %v, want it to wrap %v", err, failure)
	}

	if out != "" {
		t.Errorf("output = %q, want nothing printed", out)
	}
}

func TestDoctorCommandTakesNoArguments(t *testing.T) {
	if _, err := run(t, &fakeDiagnose{}, "extra"); err == nil {
		t.Error("Execute with an argument = nil error, want an error")
	}
}

func TestDoctorCommandDiagnosesTheWorkingDirectory(t *testing.T) {
	want, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	diagnose := &fakeDiagnose{}

	if _, err := run(t, diagnose); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if diagnose.gotDir != want {
		t.Errorf("diagnosed %q, want %q", diagnose.gotDir, want)
	}
}

func TestSectionHeader(t *testing.T) {
	for _, tc := range []struct {
		name    string
		section domain.Section
		want    string
	}{
		{
			name: "a clean section with nothing to report",
			section: domain.Section{
				Category: domain.CategorySettings,
				Results:  []domain.Result{domain.CodefallDir.Pass(), domain.SettingsComplete.Pass()},
			},
			want: "[✓] Settings",
		},
		{
			name: "the passing details, joined",
			section: domain.Section{
				Category: domain.CategoryGitHub,
				Results: []domain.Result{
					domain.GHInstalled.PassWithDetail("gh version 2.97.0 (2026-07-31)"),
					domain.GHAuthenticated.PassWithDetail("logged in as djensen47"),
					domain.GHScopes.Pass(),
				},
			},
			want: "[✓] GitHub CLI (gh version 2.97.0 (2026-07-31), logged in as djensen47)",
		},
		{
			name: "a warning takes the mark, and the passing details stay",
			section: domain.Section{
				Category: domain.CategoryBeads,
				Results: []domain.Result{
					domain.BeadsInstalled.PassWithDetail("bd version 1.2.2 (Homebrew)"),
					domain.BeadsInitialized.Warn("no database", mo.Some("bd init")),
				},
			},
			want: "[!] Beads (bd version 1.2.2 (Homebrew))",
		},
		{
			name: "a failure takes the mark, and only passing details are parenthesised",
			section: domain.Section{
				Category: domain.CategorySettings,
				Results: []domain.Result{
					domain.CodefallDir.Pass(),
					domain.SettingsFile.Fail(".codefall/settings.json not found", mo.None[string]()),
				},
			},
			want: "[✗] Settings",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Only the mark and the parenthetical are styled; strip the styling to compare the text.
			if got := stripANSI(sectionHeader(tc.section)); got != tc.want {
				t.Errorf("sectionHeader() = %q, want %q", got, tc.want)
			}
		})
	}
}

// The brackets belong to the mark rather than to the line around it: all three characters are
// rendered by one style, so a header opens with a single coloured token.
func TestSectionHeaderStylesTheBracketsWithTheMark(t *testing.T) {
	for _, tc := range []struct {
		name    string
		section domain.Section
		status  domain.Status
		want    string
	}{
		{
			name: "a passing section",
			section: domain.Section{
				Category: domain.CategorySettings,
				Results:  []domain.Result{domain.CodefallDir.Pass()},
			},
			status: domain.StatusPass,
			want:   "[✓]",
		},
		{
			name: "a warning",
			section: domain.Section{
				Category: domain.CategoryBeads,
				Results:  []domain.Result{domain.BeadsInitialized.Warn("no database", mo.Some("bd init"))},
			},
			status: domain.StatusWarn,
			want:   "[!]",
		},
		{
			name: "a failure",
			section: domain.Section{
				Category: domain.CategorySettings,
				Results: []domain.Result{
					domain.SettingsFile.Fail(".codefall/settings.json not found", mo.None[string]()),
				},
			},
			status: domain.StatusFail,
			want:   "[✗]",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := statusStyle(tc.status).Render(tc.want)

			if got := sectionHeader(tc.section); !strings.HasPrefix(got, want+" ") {
				t.Errorf("sectionHeader() = %q, want it to open with the styled token %q", got, want)
			}
		})
	}
}

func TestProblemLine(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result domain.Result
		want   string
	}{
		{
			name:   "a warning",
			result: domain.BeadsInitialized.Warn("This repository has no Beads database", mo.Some("bd init")),
			want:   "! This repository has no Beads database",
		},
		{
			name:   "a failure",
			result: domain.SettingsJSON.Fail("settings.json is not valid JSON at byte 4: x", mo.None[string]()),
			want:   "✗ settings.json is not valid JSON at byte 4: x",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripANSI(problemLine(tc.result)); got != tc.want {
				t.Errorf("problemLine() = %q, want %q", got, tc.want)
			}
		})
	}
}

// Two problems in one section is the shape the whole-command tests above never reach, and it is the
// only one that draws the square elbow and the vertical carried down past it.
func TestSectionTreeGuidesEveryProblemAndItsRemedy(t *testing.T) {
	section := domain.Section{
		Category: domain.CategorySettings,
		Results: []domain.Result{
			domain.CodefallDir.Pass(),
			domain.SettingsFile.Fail(".codefall/settings.json not found",
				mo.Some("create .codefall/settings.json")),
			domain.SettingsJSON.Fail("settings.json is not valid JSON at byte 4: x", mo.None[string]()),
		},
	}

	want := "[✗] Settings\n" +
		"├─ ✗ .codefall/settings.json not found\n" +
		"│  ╰─ fix: create .codefall/settings.json\n" +
		"╰─ ✗ settings.json is not valid JSON at byte 4: x"

	if got := stripANSI(sectionTree(section)); got != want {
		t.Errorf("sectionTree() =\n%q\nwant\n%q", got, want)
	}
}

// A passing section has no children, so it draws no guides at all.
func TestSectionTreeOfAPassingSectionIsItsHeaderAlone(t *testing.T) {
	section := domain.Section{
		Category: domain.CategoryBeads,
		Results: []domain.Result{
			domain.BeadsInstalled.PassWithDetail("bd version 1.2.2 (Homebrew)"),
			domain.BeadsInitialized.Pass(),
		},
	}

	want := "[✓] Beads (bd version 1.2.2 (Homebrew))"
	if got := stripANSI(sectionTree(section)); got != want {
		t.Errorf("sectionTree() = %q, want %q", got, want)
	}
}

// Dimming is invisible to the tests above, because the colorprofile writer strips it before the
// buffer sees it. Assert it against the styled strings, written to a plain buffer.
func TestSecondaryTextIsFaint(t *testing.T) {
	// faint is dim + teal-grey. The exact escape is faint (2) plus a 24-bit
	// foreground, e.g. "\x1b[2;38;2;138;163;168m" on dark. Check for the
	// faint attribute rather than an exact byte sequence so the test does not
	// pin the colour.
	const faintAttr = "\x1b[2"

	section := domain.Section{
		Category: domain.CategoryBeads,
		Results: []domain.Result{
			domain.BeadsInstalled.PassWithDetail("bd version 1.2.2 (Homebrew)"),
			domain.BeadsInitialized.Warn("This repository has no Beads database", mo.Some("bd init")),
		},
	}

	header := sectionHeader(section)
	if !strings.Contains(header, faintAttr) || !strings.Contains(header, "(bd version 1.2.2 (Homebrew))") {
		t.Errorf("sectionHeader() = %q, want the parenthetical dimmed, parentheses included", header)
	}

	// The sentence that says what is wrong is the one thing left at full strength.
	if problem := problemLine(section.Results[1]); strings.Contains(problem, faintAttr) {
		t.Errorf("problemLine() = %q, want nothing dimmed", problem)
	}

	var out bytes.Buffer

	if err := renderReport(&out, domain.NewReport(section.Results...)); err != nil {
		t.Fatalf("renderReport: %v", err)
	}

	if s := out.String(); !strings.Contains(s, faintAttr) || !strings.Contains(s, "fix: bd init") {
		t.Errorf("renderReport() =\n%q\nwant the whole fix line dimmed", s)
	}
}

// The summary carries the same colour as the mark for the status it reports, bullet and all, so the
// closing line agrees with what the report above it said.
func TestSummaryLineTakesTheColourOfTheStatusItReports(t *testing.T) {
	for _, tc := range []struct {
		name   string
		issues int
		status domain.Status
		want   string
	}{
		{name: "clean", issues: 0, status: domain.StatusPass, want: "• No issues found."},
		{name: "one category", issues: 1, status: domain.StatusWarn, want: "• Doctor found issues in 1 category."},
		{name: "several", issues: 3, status: domain.StatusWarn, want: "• Doctor found issues in 3 categories."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := summaryLine(tc.issues)

			if stripANSI(got) != tc.want {
				t.Errorf("summaryLine(%d) = %q, want the text %q", tc.issues, got, tc.want)
			}

			if want := statusStyle(tc.status).Render(tc.want); got != want {
				t.Errorf("summaryLine(%d) = %q, want it styled as %q", tc.issues, got, want)
			}
		})
	}
}

// stripANSI removes the escape sequences lipgloss renders, which the colorprofile writer would
// normally downsample away on its way to a non-terminal.
func stripANSI(s string) string {
	var b strings.Builder

	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}

			continue
		}

		b.WriteByte(s[i])
	}

	return b.String()
}
