package application

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// The tree init makes under the declared testing root (ADR-007): the cases, the project's own rules
// beside them in AGENTS.md, what the tree is in README.md, and the CLAUDE.md pointer for the harness
// that reads one.
const (
	testCasesDir = "test-cases"
	readmeName   = "README.md"

	// testingSkeletons is the directory of the embedded tree that holds the two documents init
	// writes at the testing root when they are missing, each under the name it lands with. They are
	// skeletons and not templates: written once, and never afterwards. From the moment one exists
	// it is the project's document, and `codefall-equip` is what fills the runners in (ADR-007).
	testingSkeletons = "agents/testing"
)

// testRoot is the testing root a run works with: the answer the request carries, and the format's
// default when it carries none.
//
// Presentation asks for it and refuses to guess when there is nobody to answer, so a request with no
// answer is one that was never surveyed — a rerun of a project settled before the block existed. The
// fallback is the format's own default rather than a second opinion about what it should be.
func testRoot(request Request) string {
	if request.TestDir == "" {
		return settings.DefaultTestDir
	}

	return request.TestDir
}

// testing is the sixth step of a run: it declares where the project's test cases live and makes the
// tree they live in.
//
// Both halves are write-once. The block is written when the settings do not already carry one, and
// an existing declaration is never moved — a project's cases are at the path it declared, and the
// runners beside it are codefall-equip's. Each file is written only when it is missing, because from
// the moment it exists it is the project's document (ADR-007).
func (i *Initialize) testing(_ context.Context, request Request) (domain.StepResult, error) {
	root := testRoot(request)

	if err := settings.ValidateTestDir(root); err != nil {
		return domain.StepResult{}, err
	}

	var done []string

	declared, err := i.declareTestRoot(request.Dir, root)
	if err != nil {
		return domain.StepResult{}, err
	}

	if declared {
		done = append(done, fmt.Sprintf("declared %s/ in %s", root, settingsName))
	}

	created, err := i.writeTestingTree(request, root)
	if err != nil {
		return domain.StepResult{}, err
	}

	if len(created) > 0 {
		done = append(done, "created "+sentenceList(created))
	}

	if len(done) == 0 {
		return domain.TestingStep.Skipped(fmt.Sprintf("the testing tree at %s/ is current", root)), nil
	}

	return domain.TestingStep.Done(sentenceList(done)), nil
}

// declareTestRoot puts the test block in the settings file and reports whether it had to, leaving a
// declaration that is already there exactly as it is.
//
// The block is spliced into the file's text rather than encoded with the rest of the document, and
// that is deliberate: this is the one step that writes settings a run did not itself write. A rerun
// on a project settled before the block existed is how most projects get theirs, and decoding that
// file and encoding it again would reorder every key and drop whatever a project or codefall-equip
// had added. Splicing keeps every other byte where it was.
func (i *Initialize) declareTestRoot(dir, root string) (bool, error) {
	data, err := i.files.ReadFile(settingsPath(dir))
	if err != nil {
		return false, fmt.Errorf("read %s: %w", settingsName, err)
	}

	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return false, fmt.Errorf("decode %s: %w", settingsName, err)
	}

	if _, declared := document[settings.BlockTest]; declared {
		return false, nil
	}

	if err := i.files.WriteFile(settingsPath(dir), []byte(withTestBlock(string(data), root))); err != nil {
		return false, fmt.Errorf("write %s: %w", settingsName, err)
	}

	return true, nil
}

// withTestBlock is the splice: the document's own text, then the block as the last member of the
// top-level object, indented the way init writes the file.
//
// The caller has already decoded the text into an object, so the last character that is not space is
// the closing brace and what comes before it is either nothing or the last member.
func withTestBlock(body, root string) string {
	const indent = "  "

	// json.Marshal on a string is the one thing that quotes and escapes a path the way JSON reads it.
	quoted, err := json.Marshal(root)
	if err != nil {
		// A string always marshals. The fallback keeps the function total rather than adding an error
		// return no caller could act on.
		quoted = []byte(`"` + settings.DefaultTestDir + `"`)
	}

	block := fmt.Sprintf("%q: {\n%s%s%q: %s\n%s}", settings.BlockTest,
		indent, indent, settings.FieldTestDir, quoted, indent)

	// One closing brace comes off, not every one the document happens to end with: a nested block is
	// the last thing a settings file says, and its brace is the file's own.
	trimmed := strings.TrimRight(body, " \t\r\n")
	head := strings.TrimRight(trimmed[:len(trimmed)-1], " \t\r\n")

	if head == "{" {
		return "{\n" + indent + block + "\n}\n"
	}

	return head + ",\n" + indent + block + "\n}\n"
}

// writeTestingTree makes the directories and writes the documents that are missing, and returns the
// files it wrote, by the path a person reads them at.
//
// The directories are made whatever the step finds, because making one that is already there does
// nothing and the file system has no cheaper way to answer the question. What the step reports is
// the files it wrote, which is what a reader can go and look at.
func (i *Initialize) writeTestingTree(request Request, root string) ([]string, error) {
	for _, dir := range []string{root, path.Join(root, testCasesDir)} {
		if err := i.files.MkdirAll(filepath.Join(request.Dir, filepath.FromSlash(dir))); err != nil {
			return nil, fmt.Errorf("create %s/: %w", dir, err)
		}
	}

	documents := make([]struct {
		name string
		body string
	}, 0, 3)

	for _, skeleton := range []struct {
		name   string
		source string
	}{
		{agentsName, testingSkeletons + "/" + agentsName},
		{readmeName, testingSkeletons + "/" + readmeName},
	} {
		body, err := i.source.Read(skeleton.source)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", skeleton.source, err)
		}

		documents = append(documents, struct {
			name string
			body string
		}{skeleton.name, string(body)})
	}

	// The pointer is the same one line the root of the project gets, and for the same reason: Claude
	// Code reads CLAUDE.md, and the rules are not kept in two places.
	if slices.Contains(chosen(request), harness.Claude) {
		documents = append(documents, struct {
			name string
			body string
		}{claudeMemoryName, claudePointer})
	}

	var created []string

	for _, document := range documents {
		at := path.Join(root, document.name)

		existing, err := i.readProjectFile(request.Dir, filepath.FromSlash(at))
		if err != nil {
			return nil, err
		}

		if existing.IsPresent() {
			continue
		}

		if err := i.files.WriteFile(
			filepath.Join(request.Dir, filepath.FromSlash(at)), []byte(document.body),
		); err != nil {
			return nil, fmt.Errorf("write %s: %w", at, err)
		}

		created = append(created, at)
	}

	return created, nil
}
