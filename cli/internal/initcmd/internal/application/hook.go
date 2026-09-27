package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
)

// commandKey is the one key every harness's hook format shares: the shell command to run. The merge
// dedupes on it, so a rerun does not repeat a registration.
const commandKey = "command"

// scriptPrefix is what every script the extension ships is named with, and what makes codefall's
// own registration recognisable in a file it shares with the project's hooks. It is a rule about
// the tree's file names, stated in extensions/AGENTS.md, and this is what depends on it.
const scriptPrefix = "codefall-"

// How a hook definition lands. The JSON harnesses get one recursive merge each — what differs
// between them is the destination, not the algorithm — and OpenCode's plugin is a copied file.
const (
	formatMerge hookFormat = iota
	formatCopy
)

type hookFormat int

// hookSpec is one harness's registration: which file in the embedded tree, where it goes in the
// project, and how to treat it.
type hookSpec struct {
	source string
	dest   string
	format hookFormat
}

// hookSpecs is the one table of what init registers where. A harness with no entry is skipped
// rather than errored: the run reports what it did not need to do. Adding a harness that reads one
// of these formats is one row; adding one that needs a new treatment is a third value of hookFormat.
var hookSpecs = map[string]hookSpec{
	harness.Claude:   {source: "hooks/claude/hooks.json", dest: ".claude/settings.json", format: formatMerge},
	harness.Codex:    {source: "hooks/codex/hooks.json", dest: ".codex/hooks.json", format: formatMerge},
	harness.Agy:      {source: "hooks/agy/hooks.json", dest: ".agents/hooks.json", format: formatMerge},
	harness.OpenCode: {source: "hooks/opencode/codefall.js", dest: ".opencode/plugins/codefall.js", format: formatCopy},
}

// hook is the fourth step of a run: it takes the harness's entry in hookSpecs and lands it. The
// extension step has already copied the shared scripts the hooks point at into .codefall/, so the
// destination of any script path in a definition exists by the time this runs. The definitions
// themselves are read straight from the embedded tree and never copied: the extension step installs
// three named subtrees, and hooks/<harness>/ is not one of them.
func (i *Initialize) hook(ctx context.Context, request Request) (domain.StepResult, error) {
	if _, err := skillsDirs(request); err != nil {
		return domain.StepResult{}, err
	}

	// The path below the repository root is the same answer for every harness, and asking git for it
	// is only worth doing when a definition that names the root is about to be registered.
	prefix := mo.None[string]()

	results := make([]domain.StepResult, 0, len(request.Harnesses))

	for _, name := range chosen(request) {
		spec, known := hookSpecs[name]
		if !known {
			results = append(results,
				domain.HookStep.Skipped(fmt.Sprintf("codefall has no hooks for %s", name)))

			continue
		}

		if spec.format == formatCopy {
			result, err := i.copyHook(request.Dir, spec)
			if err != nil {
				return domain.StepResult{}, err
			}

			results = append(results, result)

			continue
		}

		if prefix.IsAbsent() {
			below, err := i.repositoryPrefix(ctx, request.Dir)
			if err != nil {
				return domain.StepResult{}, err
			}

			prefix = mo.Some(below)
		}

		result, err := i.mergeHook(request.Dir, prefix.OrEmpty(), spec)
		if err != nil {
			return domain.StepResult{}, err
		}

		results = append(results, result)
	}

	return oneHookResult(results), nil
}

// oneHookResult is what the step reports when it has registered with several harnesses: the outcome
// is Done when any registration changed something, and the detail is every harness's own sentence.
// A run for one harness reports that harness's result as it is, which is what the step has always
// said.
func oneHookResult(results []domain.StepResult) domain.StepResult {
	switch len(results) {
	case 0:
		return domain.HookStep.Skipped("no harness to register codefall's hooks with")
	case 1:
		return results[0]
	}

	details := make([]string, 0, len(results))
	done := false

	for _, result := range results {
		details = append(details, result.Detail)

		if result.Outcome == domain.OutcomeDone {
			done = true
		}
	}

	detail := strings.Join(details, "; ")

	if done {
		return domain.HookStep.Done(detail)
	}

	return domain.HookStep.Skipped(detail)
}

// mergeHook merges the harness's hook definitions into the file it reads them from. The
// destination is decoded as a plain object so everything else it says — permissions, plugin
// declarations, other hooks — survives the round trip; encoding/json sorts keys on the way out,
// which is a one-time diff on hand-ordered files.
//
// prefix is dir's path below the repository root, empty at the root. A definition that names the
// root has it written in, so a subdirectory install points at the scripts it copied rather than at a
// root that has none.
func (i *Initialize) mergeHook(dir, prefix string, spec hookSpec) (domain.StepResult, error) {
	source, err := i.source.Read(spec.source)
	if err != nil {
		return domain.StepResult{}, fmt.Errorf("read %s: %w", spec.source, err)
	}

	var addition map[string]any
	if err := json.Unmarshal(source, &addition); err != nil {
		return domain.StepResult{}, fmt.Errorf("decode %s: %w", spec.source, err)
	}

	placeUnder(addition, prefix)

	body, err := i.readDestination(filepath.Join(dir, spec.dest), spec.dest)
	if err != nil {
		return domain.StepResult{}, err
	}

	document := map[string]any{}
	if data, said := body.Get(); said {
		if err := json.Unmarshal(data, &document); err != nil {
			return domain.StepResult{}, fmt.Errorf("decode %s: %w", spec.dest, err)
		}
	}

	changed, err := mergeInto(document, addition, "", spec.dest)
	if err != nil {
		return domain.StepResult{}, err
	}

	if !changed {
		return domain.HookStep.Skipped(fmt.Sprintf("codefall's hooks are already in %s", spec.dest)), nil
	}

	data, err := encodeDocument(document, spec.dest)
	if err != nil {
		return domain.StepResult{}, err
	}

	if err := i.files.MkdirAll(filepath.Join(dir, filepath.Dir(spec.dest))); err != nil {
		return domain.StepResult{}, fmt.Errorf("create %s/: %w", filepath.Dir(spec.dest), err)
	}

	if err := i.files.WriteFile(filepath.Join(dir, spec.dest), data); err != nil {
		return domain.StepResult{}, fmt.Errorf("write %s: %w", spec.dest, err)
	}

	return domain.HookStep.Done(fmt.Sprintf("merged codefall's hooks into %s", spec.dest)), nil
}

// copyHook installs the harness's plugin by copying one file over. A plugin whose bytes match is a
// skip: what would be written is what is there.
func (i *Initialize) copyHook(dir string, spec hookSpec) (domain.StepResult, error) {
	source, err := i.source.Read(spec.source)
	if err != nil {
		return domain.StepResult{}, fmt.Errorf("read %s: %w", spec.source, err)
	}

	path := filepath.Join(dir, spec.dest)

	current, err := i.files.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return domain.StepResult{}, fmt.Errorf("read %s: %w", spec.dest, err)
	}
	if err == nil && bytes.Equal(current, source) {
		return domain.HookStep.Skipped(fmt.Sprintf("codefall's plugin at %s is already installed", spec.dest)), nil
	}

	if err := i.files.MkdirAll(filepath.Join(dir, filepath.Dir(spec.dest))); err != nil {
		return domain.StepResult{}, fmt.Errorf("create %s/: %w", filepath.Dir(spec.dest), err)
	}

	if err := i.files.WriteFile(path, source); err != nil {
		return domain.StepResult{}, fmt.Errorf("write %s: %w", spec.dest, err)
	}

	return domain.HookStep.Done(fmt.Sprintf("installed codefall's plugin at %s", spec.dest)), nil
}

// mergeInto folds the addition into the document and reports whether anything came in. Objects
// merge recursively and arrays are appended; a scalar the document already says is kept, because
// only the lists the file holds are codefall's to grow — an Antigravity hook the project disabled
// stays disabled. A path where the two disagree about type is an error naming it: the file is not
// shaped the way the harness expects, and init is not the thing that should decide what it meant.
func mergeInto(document, addition map[string]any, path string, dest string) (bool, error) {
	changed := false

	for key, incoming := range addition {
		existing, declared := document[key]
		// A key holding null says nothing, the same as a key that is absent: a hand-edited file
		// that nulls out a section it does not use is not a file shaped against the harness.
		if !declared || existing == nil {
			document[key] = incoming
			changed = true
			continue
		}

		merged, grew, err := mergeValue(existing, incoming, join(path, key), dest)
		if err != nil {
			return false, err
		}
		if grew {
			document[key] = merged
			changed = true
		}
	}

	return changed, nil
}

// mergeValue resolves one key's worth of source against the same key in the destination. Scalars
// and source entries that turn up nothing new never change the destination.
func mergeValue(existing, incoming any, path, dest string) (any, bool, error) {
	switch src := incoming.(type) {
	case map[string]any:
		dst, ok := existing.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("%s: %s is not an object", dest, path)
		}
		grew, err := mergeInto(dst, src, path, dest)
		if err != nil {
			return nil, false, err
		}
		return dst, grew, nil

	case []any:
		dst, ok := existing.([]any)
		if !ok {
			return nil, false, fmt.Errorf("%s: %s is not an array", dest, path)
		}
		merged, grew := appendEntries(dst, src)
		return merged, grew, nil
	}

	// A scalar the destination already set is the destination's own business — but a destination
	// holding an object or an array where the definition has a scalar is a file shaped against the
	// harness, which is the same complaint the two cases above make from the other side.
	switch existing.(type) {
	case map[string]any, []any:
		return nil, false, fmt.Errorf("%s: %s is not a scalar", dest, path)
	}

	return nil, false, nil
}

// appendEntries lands each source entry the destination does not already run, and reports whether
// the array changed. An entry is known by its matcher together with the commands it runs: the same
// command registered under another matcher still leaves this event unguarded, so it joins rather
// than being taken for present. The command is wherever the harness's format keeps it — inside a
// nested handler list, as in Claude's and Codex's, or on the entry itself, as in Antigravity's — so
// the search is recursive. An entry that declares no command is joined by content instead: the same
// entry twice is one entry.
//
// Three things happen before an entry is appended, in this order.
//
// An entry naming a codefall script replaces the destination's entry naming that same script under
// the same matcher, rather than joining it. The command string alone cannot say that: a renamed
// script, a new flag, or a project that moved below the repository root all change it, and an
// append would leave the old registration running beside the new one with nothing able to remove
// it. The script's own name is the identity, which is what the codefall- prefix is for.
//
// An entry that matches everything is already covered by any entry running its commands under a
// narrower matcher. A project that primes Beads on startup alone has said what it wants, and
// adding the match-all entry beside it would prime on every other session source and twice on
// that one.
//
// An entry whose commands are only partly new joins with only the new ones. Appending it whole
// would run the command the destination already had twice over.
func appendEntries(dst, src []any) ([]any, bool) {
	known := map[string]bool{}
	runs := map[string]bool{}

	for _, entry := range dst {
		for _, command := range commandsIn(entry) {
			known[matcherOf(entry)+"\x00"+command] = true
			runs[command] = true
		}
	}

	changed := false

	for _, entry := range src {
		commands := commandsIn(entry)

		if len(commands) == 0 {
			duplicate := false
			for _, existing := range dst {
				if reflect.DeepEqual(existing, entry) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				dst = append(dst, entry)
				changed = true
			}
			continue
		}

		matcher := matcherOf(entry)

		if at, found := ownEntry(dst, entry, matcher); found {
			if !reflect.DeepEqual(dst[at], entry) {
				dst[at] = entry
				changed = true
			}

			record(known, runs, matcher, commands)

			continue
		}

		if matcher == "" && coveredBy(runs, commands) {
			continue
		}

		joining, fresh := newCommands(entry, known, matcher)
		if len(fresh) == 0 {
			continue
		}

		dst = append(dst, joining)
		changed = true
		record(known, runs, matcher, fresh)
	}

	return dst, changed
}

// record notes that a matcher now runs these commands, for the entries still to come.
func record(known, runs map[string]bool, matcher string, commands []string) {
	for _, command := range commands {
		known[matcher+"\x00"+command] = true
		runs[command] = true
	}
}

// ownEntry finds the destination's own registration of the same codefall script under the same
// matcher, which is the entry an upgrade replaces. An entry naming no codefall script is nobody's
// to replace: the destination's other hooks are the project's.
func ownEntry(dst []any, entry any, matcher string) (int, bool) {
	scripts := scriptsIn(entry)
	if len(scripts) == 0 {
		return 0, false
	}

	for at, existing := range dst {
		if matcherOf(existing) != matcher {
			continue
		}

		for _, script := range scriptsIn(existing) {
			if slices.Contains(scripts, script) {
				return at, true
			}
		}
	}

	return 0, false
}

// coveredBy reports whether every command is already run somewhere in this event, whatever matcher
// it runs under.
func coveredBy(runs map[string]bool, commands []string) bool {
	for _, command := range commands {
		if !runs[command] {
			return false
		}
	}

	return true
}

// newCommands is the entry to append and the commands it brings: the whole entry when all of them
// are new, and a copy holding only the handlers the destination does not run when some are. An
// entry that keeps its command on itself rather than in a handler list cannot be split, so it
// joins whole or not at all.
func newCommands(entry any, known map[string]bool, matcher string) (any, []string) {
	var fresh []string

	for _, command := range commandsIn(entry) {
		if !known[matcher+"\x00"+command] {
			fresh = append(fresh, command)
		}
	}

	object, ok := entry.(map[string]any)
	if !ok || len(fresh) == 0 || len(fresh) == len(commandsIn(entry)) {
		return entry, fresh
	}

	handlers, ok := object["hooks"].([]any)
	if !ok {
		return entry, fresh
	}

	joining := make([]any, 0, len(handlers))
	for _, handler := range handlers {
		for _, command := range commandsIn(handler) {
			if !known[matcher+"\x00"+command] {
				joining = append(joining, handler)
				break
			}
		}
	}

	split := maps.Clone(object)
	split["hooks"] = joining

	return split, fresh
}

// scriptsIn is every codefall script an entry's commands name, by the script's own file name. A
// command names one wherever the path it holds happens to point: the definitions reach the same
// script through the repository root, through the workspace, and through the plugin's own
// directory, and an upgrade changes which.
func scriptsIn(entry any) []string {
	var scripts []string

	for _, command := range commandsIn(entry) {
		if script, found := scriptIn(command); found {
			scripts = append(scripts, script)
		}
	}

	return scripts
}

// scriptIn takes the codefall script's file name out of one command, if it names one. What follows
// the prefix is the file name's own characters, so the quote, the flag, and the rest of the command
// line all end it.
func scriptIn(command string) (string, bool) {
	at := strings.Index(command, scriptPrefix)
	if at < 0 {
		return "", false
	}

	name := command[at:]
	for offset, character := range name {
		if !scriptNameCharacter(character) {
			return name[:offset], true
		}
	}

	return name, true
}

func scriptNameCharacter(character rune) bool {
	switch {
	case character >= 'a' && character <= 'z',
		character >= 'A' && character <= 'Z',
		character >= '0' && character <= '9':
		return true
	default:
		return character == '-' || character == '_' || character == '.'
	}
}

// matcherOf is the matcher's half of an entry's identity: the filter the harness applies before the
// entry's command ever runs. An entry that declares none matches everything, which is the empty
// string — the same nothing a missing matcher means to the harness.
func matcherOf(entry any) string {
	object, ok := entry.(map[string]any)
	if !ok {
		return ""
	}

	matcher, _ := object["matcher"].(string)
	return matcher
}

// join names the path a reader would go and look at, rather than whichever key a helper happens to
// know the name of.
func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// commandsIn collects every command string an entry runs, whatever shape the entry has. An entry
// that holds none contributes nothing to the dedupe set.
func commandsIn(value any) []string {
	switch shaped := value.(type) {
	case map[string]any:
		var commands []string
		if command, ok := shaped[commandKey].(string); ok {
			commands = append(commands, command)
		}
		for _, nested := range shaped {
			commands = append(commands, commandsIn(nested)...)
		}
		return commands

	case []any:
		var commands []string
		for _, nested := range shaped {
			commands = append(commands, commandsIn(nested)...)
		}
		return commands
	}

	return nil
}

// readDestination is the one read of a file a merge writes back: three files say the same nothing —
// absent, empty or whitespace, and the JSON literal null — and they are one answer here rather than
// three behaviours further down, because json.Unmarshal accepts null into anything while rejecting
// the other two (ADR-GO-03). A read failure names the destination the way a person reads it in a
// report, the same display form every other error in the step uses.
func (i *Initialize) readDestination(path, display string) (mo.Option[[]byte], error) {
	data, err := i.files.ReadFile(path)

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return mo.None[[]byte](), nil
	case err != nil:
		return mo.None[[]byte](), fmt.Errorf("read %s: %w", display, err)
	}

	if trimmed := strings.TrimSpace(string(data)); trimmed == "" || trimmed == "null" {
		return mo.None[[]byte](), nil
	}

	return mo.Some(data), nil
}

// encodeDocument renders the merged document back into the bytes that go in the file. The encoder
// is built by hand rather than through json.Marshal because Marshal escapes <, > and & into \u
// sequences, which would silently rewrite a permission rule like "Bash(a && b)" in a file codefall
// does not own. Encode writes the trailing newline itself.
func encodeDocument(document map[string]any, dest string) ([]byte, error) {
	var buffer bytes.Buffer

	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(document); err != nil {
		return nil, fmt.Errorf("encode %s: %w", dest, err)
	}

	return buffer.Bytes(), nil
}
