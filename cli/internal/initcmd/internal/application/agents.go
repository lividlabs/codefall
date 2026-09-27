package application

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
)

// The two files this step knows about, named as a person reads them. AGENTS.md holds the rules;
// CLAUDE.md is a pointer to it, because Claude Code reads CLAUDE.md and the rules are not kept in
// two places. The pointer is the whole file, one line and a newline.
const (
	agentsName       = "AGENTS.md"
	claudeMemoryName = "CLAUDE.md"
	claudePointer    = "See [AGENTS.md](AGENTS.md) — the rules for this repo live there, and there only.\n"
)

// section is one of codefall's marked sections of AGENTS.md: the name a person reads it by, the
// markers that delimit it wherever it has been written, and the words themselves.
type section struct {
	name  string
	begin string
	end   string
	body  string
}

// sectionSpec is where one section's words are in the embedded tree, with the markers the domain
// says delimit it. The words are read from the tree the way the hook step reads a harness's
// definition: content the binary ships and init splices into a project's file rather than copies.
type sectionSpec struct {
	name   string
	begin  string
	end    string
	source string
}

// sectionsSource is the directory of the embedded tree that holds the sections, one file each. The
// extension step copies nothing from under agents/: what is there is read here and written into
// AGENTS.md, and a project never receives the files themselves.
const sectionsSource = "agents/sections"

// sectionSpecs is every section the step writes, in the order a file that has none gains them. Each
// pair of markers is its own, so a project that has one section and not the others gains the ones
// it is missing beside the one it has.
//
// The Codefall section goes first because it is the frame the other three sit inside: the chain of
// verbs, and who is authoritative for what. A file that already has the other three gains it at the
// end, after them, because the step never moves what somebody else wrote.
var sectionSpecs = []sectionSpec{
	{name: "Codefall", begin: domain.CodefallSectionBegin, end: domain.CodefallSectionEnd, source: sectionsSource + "/codefall.md"},
	{name: "Beads", begin: domain.BeadsSectionBegin, end: domain.BeadsSectionEnd, source: sectionsSource + "/beads.md"},
	{name: "Local environment", begin: domain.LocalSectionBegin, end: domain.LocalSectionEnd, source: sectionsSource + "/local.md"},
	{name: "Testing", begin: domain.TestingSectionBegin, end: domain.TestingSectionEnd, source: sectionsSource + "/testing.md"},
}

// sectionsFor reads every section out of the tree, in the order the step writes them, with the
// testing root filled in. It takes the root because the last section names it: the other three are
// the same words in every project, and the path a project's cases live at is the project's own
// (ADR-007). The placeholder is filled in every section rather than one, so a section that comes
// to name the root later needs no change here.
//
// A section the tree does not hold, or one whose markers are not where the splice expects them, is
// an error naming the file: both are a binary shipped wrong, and neither is worth guessing around.
func (i *Initialize) sectionsFor(root string) ([]section, error) {
	sections := make([]section, 0, len(sectionSpecs))

	for _, spec := range sectionSpecs {
		data, err := i.source.Read(spec.source)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", spec.source, err)
		}

		body := strings.ReplaceAll(string(data), domain.TestingRootPlaceholder, root)

		if err := marked(body, spec); err != nil {
			return nil, err
		}

		sections = append(sections, section{name: spec.name, begin: spec.begin, end: spec.end, body: body})
	}

	return sections, nil
}

// marked checks a section against what the splice expects of it: one pair of markers, the opening
// one the first line and the closing one the last, with the newline that ends the file after it. A
// second marker in the body would make a later run replace the wrong span, and a section without
// its pair would be written once and never found again.
func marked(body string, spec sectionSpec) error {
	switch {
	case !strings.HasPrefix(body, spec.begin+"\n"):
		return fmt.Errorf("%s does not open with %s", spec.source, spec.begin)
	case !strings.HasSuffix(body, "\n"+spec.end+"\n"):
		return fmt.Errorf("%s does not close with %s and a newline", spec.source, spec.end)
	case strings.Count(body, spec.begin) != 1 || strings.Count(body, spec.end) != 1:
		return fmt.Errorf("%s holds a marker more than once", spec.source)
	}

	return nil
}

// sectionChange is what the step did to one section, which is most of what it has to report.
type sectionChange int

const (
	sectionCurrent sectionChange = iota
	sectionAdded
	sectionReplaced
)

// agents is the fifth step of a run: it writes codefall's own sections into AGENTS.md, and gives
// Claude Code the CLAUDE.md pointer to it when the project has no CLAUDE.md at all.
//
// bd would have written a section of its own here, and init runs `bd init --skip-agents` because
// that text describes a setup codefall does not create. So the sections are codefall's, and this
// is what writes them.
//
// It runs after the Beads step on purpose. bd init stages AGENTS.md and CLAUDE.md when they exist
// and commits what it staged under its own message, so an edit made before it ran would land in
// bd's commit rather than the author's.
func (i *Initialize) agents(_ context.Context, request Request) (domain.StepResult, error) {
	done, err := i.writeSections(request.Dir, testRoot(request))
	if err != nil {
		return domain.StepResult{}, err
	}

	pointed, err := i.writeClaudePointer(request)
	if err != nil {
		return domain.StepResult{}, err
	}

	if pointed {
		done = append(done, "created "+claudeMemoryName)
	}

	// Nothing done is the skip: every section was already what codefall would have written, and
	// CLAUDE.md was not this run's to make.
	if len(done) == 0 {
		return domain.AgentsStep.Skipped(
			fmt.Sprintf("codefall's sections in %s are current", agentsName)), nil
	}

	return domain.AgentsStep.Done(sentenceList(done)), nil
}

// writeSections puts codefall's sections in AGENTS.md and reports what that took, as the clauses of
// the sentence the step says about it — none when the file already said everything.
//
// The file belongs to the project, so the step changes as little of it as it can: a section already
// marked is replaced between its markers and everything on either side survives byte for byte, and
// a file with no markers for a section keeps what it says and gains that section at the end. A file
// that is already what would be written is not written at all, which is what makes a second run a
// skip. The file is read once and written once, however many sections change.
func (i *Initialize) writeSections(dir, root string) ([]string, error) {
	sections, err := i.sectionsFor(root)
	if err != nil {
		return nil, err
	}

	body, err := i.readProjectFile(dir, agentsName)
	if err != nil {
		return nil, err
	}

	existing, present := body.Get()
	next := existing

	changes := make([]sectionChange, 0, len(sections))

	for _, s := range sections {
		var change sectionChange

		next, change, err = sectionIn(next, s)
		if err != nil {
			return nil, err
		}

		changes = append(changes, change)
	}

	if next == existing {
		return nil, nil
	}

	if err := i.files.WriteFile(filepath.Join(dir, agentsName), []byte(next)); err != nil {
		return nil, fmt.Errorf("write %s: %w", agentsName, err)
	}

	return sectionsDone(changes, present, sections), nil
}

// sectionsDone is what happened to the sections, one clause each. A file that was not there was
// created with every section; otherwise the sections that changed are named with what happened to
// them, in section order.
func sectionsDone(changes []sectionChange, present bool, sections []section) []string {
	if !present {
		names := make([]string, 0, len(sections))
		for _, s := range sections {
			names = append(names, s.name)
		}

		return []string{fmt.Sprintf("created %s with %s", agentsName, sectionPhrase(names))}
	}

	var added, replaced []string

	for at, change := range changes {
		switch change {
		case sectionAdded:
			added = append(added, sections[at].name)
		case sectionReplaced:
			replaced = append(replaced, sections[at].name)
		case sectionCurrent:
		}
	}

	var done []string

	if len(replaced) > 0 {
		done = append(done, fmt.Sprintf("updated %s in %s", sectionPhrase(replaced), agentsName))
	}

	if len(added) > 0 {
		done = append(done, fmt.Sprintf("added %s to %s", sectionPhrase(added), agentsName))
	}

	return done
}

// sectionPhrase names some sections the way a sentence does: "the Beads section", or "the Beads and
// Local environment sections".
func sectionPhrase(names []string) string {
	if len(names) == 1 {
		return "the " + names[0] + " section"
	}

	return "the " + sentenceList(names) + " sections"
}

// sectionIn is the string surgery: what the file should say, given what it says now and one section.
// The encoding of a section is here rather than in the domain, which holds the words and the
// markers and nothing about how they are spliced into somebody's file.
func sectionIn(existing string, s section) (string, sectionChange, error) {
	// The file in the tree ends with a newline after its closing marker. What is spliced into a
	// file does not carry it, because where it lands decides what follows.
	body := strings.TrimSuffix(s.body, "\n")

	begin := strings.Index(existing, s.begin)
	if begin >= 0 {
		// A file with an opening marker and no closing one cannot be edited without guessing where
		// codefall's words stop and the project's resume, and a guess here silently eats somebody's
		// prose.
		offset := strings.Index(existing[begin:], s.end)
		if offset < 0 {
			return "", sectionCurrent, fmt.Errorf("%s has %s with no %s after it", agentsName, s.begin, s.end)
		}

		end := begin + offset + len(s.end)

		next := existing[:begin] + body + existing[end:]
		if next == existing {
			return existing, sectionCurrent, nil
		}

		return next, sectionReplaced, nil
	}

	// One blank line between what the file said and the section, however the file ended, and a
	// single newline at the end of it.
	next := body + "\n"
	if trimmed := strings.TrimRight(existing, "\n"); trimmed != "" {
		next = trimmed + "\n\n" + next
	}

	return next, sectionAdded, nil
}

// writeClaudePointer creates CLAUDE.md for the harness that reads it, and reports whether it did.
//
// Only when the file is missing. A CLAUDE.md the project already has says whatever its author meant
// it to say, and replacing that with a pointer would throw the rules away rather than point at them.
func (i *Initialize) writeClaudePointer(request Request) (bool, error) {
	if !slices.Contains(chosen(request), harness.Claude) {
		return false, nil
	}

	body, err := i.readProjectFile(request.Dir, claudeMemoryName)
	if err != nil {
		return false, err
	}

	if body.IsPresent() {
		return false, nil
	}

	if err := i.files.WriteFile(
		filepath.Join(request.Dir, claudeMemoryName), []byte(claudePointer),
	); err != nil {
		return false, fmt.Errorf("write %s: %w", claudeMemoryName, err)
	}

	return true, nil
}

// readProjectFile reads one of the project's markdown files. A file that is not there is None rather
// than an error, because this step creates what it needs (ADR-GO-03); a file that cannot be read is
// an error naming it, because the step was about to write it.
//
// Unlike the harness's settings, an empty file is present and empty: markdown has no shape to be
// wrong, and a file somebody has made is a file the step appends to rather than creates.
func (i *Initialize) readProjectFile(dir, name string) (mo.Option[string], error) {
	data, err := i.files.ReadFile(filepath.Join(dir, name))

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return mo.None[string](), nil
	case err != nil:
		return mo.None[string](), fmt.Errorf("read %s: %w", name, err)
	}

	return mo.Some(string(data)), nil
}
