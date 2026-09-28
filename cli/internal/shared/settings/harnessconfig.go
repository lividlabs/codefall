package settings

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
)

// How each harness is called when an agent runs on it. A harness name with no block runs bare, as
// it always has. A block under a harness's own name configures that harness; a block under any
// other key is a variant, and its harness field names the binary. An agent in a list names either
// one, so one project can call Codex through a provider for review and directly for consult. The
// shared script that starts a harness reads the block at run time; this module stores it, checks
// it, and says which binary a name resolves to.
const (
	// FieldHarnessConfig is the top-level object, keyed by harness name or variant key.
	FieldHarnessConfig = "harnessConfig"
	// FieldConfigHarness is a variant block's binary: one of the five harness names, never current.
	// A block keyed by a harness name must not carry it, since the key already says.
	FieldConfigHarness = "harness"
	// FieldModelFlag is the flag the shared script passes the model with. The script's built-in flag
	// for the harness is the fallback.
	FieldModelFlag = "modelFlag"
	// FieldProvider is the harness's own provider name, which the script turns into that harness's
	// switch.
	FieldProvider = "provider"
	// FieldArgs is extra command-line arguments, appended verbatim.
	FieldArgs = "args"
	// FieldEnv is a shell command whose standard output is evaluated before the harness starts, so
	// export lines take effect.
	FieldEnv = "env"
	// HarnessKeyPattern is how a harnessConfig key may be written: lowercase letters and digits in
	// hyphen-separated runs, the shape every harness name already has, so a key reads the same
	// wherever it appears, in the file, on a command line, and after the colon in `via=`.
	HarnessKeyPattern = `^[a-z0-9]+(-[a-z0-9]+)*$`
)

var harnessKeyRegexp = regexp.MustCompile(HarnessKeyPattern)

// HarnessConfig is one block, read out of a document that Validate accepts: the binary it runs, and
// what the shared script adds to the call. Harness is the key when the key is a harness name.
type HarnessConfig struct {
	Harness   string
	ModelFlag mo.Option[string]
	Provider  mo.Option[string]
	Env       mo.Option[string]
	Args      []string
}

// HarnessConfigFields returns the fields a block may carry, in the order the file writes them.
func HarnessConfigFields() []string {
	return []string{FieldConfigHarness, FieldModelFlag, FieldProvider, FieldArgs, FieldEnv}
}

// HarnessConfigs returns the blocks a document defines, keyed as the file keys them. A block
// Validate would reject is left out, like the other readers do: the caller has already been told
// what is wrong with it, and a block nothing can act on is no block.
func HarnessConfigs(doc Document) map[string]HarnessConfig {
	configs := map[string]HarnessConfig{}

	value, present := lookup(doc, FieldHarnessConfig)
	if !present {
		return configs
	}

	blocks, ok := value.(map[string]any)
	if !ok {
		return configs
	}

	for key, block := range blocks {
		if harnessBlockProblem(key, block) != "" {
			continue
		}

		configs[key] = harnessConfig(key, block)
	}

	return configs
}

// HarnessConfigKeys returns the keys a document's accepted blocks are under, sorted.
func HarnessConfigKeys(doc Document) []string {
	return slices.Sorted(maps.Keys(HarnessConfigs(doc)))
}

// ResolveHarness returns the binary an agent's harness field names: a harness name or current is
// itself, a block key is the block's harness, and anything else is None. It is what a check that
// looks for the binary asks before it looks.
func ResolveHarness(doc Document, key string) mo.Option[string] {
	if key == HarnessCurrent || harness.SkillsDir(key).IsPresent() {
		return mo.Some(key)
	}

	if config, ok := HarnessConfigs(doc)[key]; ok {
		return mo.Some(config.Harness)
	}

	return mo.None[string]()
}

// harnessConfig reads a block harnessBlockProblem has accepted.
func harnessConfig(key string, value any) HarnessConfig {
	fields, _ := value.(map[string]any)

	config := HarnessConfig{
		Harness:   key,
		ModelFlag: optionalText(fields, FieldModelFlag),
		Provider:  optionalText(fields, FieldProvider),
		Env:       optionalText(fields, FieldEnv),
	}

	if runs, ok := optionalText(fields, FieldConfigHarness).Get(); ok {
		config.Harness = runs
	}

	if items, ok := fields[FieldArgs].([]any); ok {
		config.Args = make([]string, 0, len(items))

		for _, item := range items {
			text, _ := item.(string)
			config.Args = append(config.Args, text)
		}
	}

	return config
}

func optionalText(fields map[string]any, name string) mo.Option[string] {
	value, present := lookup(fields, name)
	if !present {
		return mo.None[string]()
	}

	text, _ := value.(string)

	return mo.Some(text)
}

// harnessConfigProblems checks every block of a harnessConfig object the top level has accepted,
// one problem per block, in key order so the report is the same on every run.
func harnessConfigProblems(doc Document) []string {
	value, present := lookup(doc, FieldHarnessConfig)
	if !present {
		return nil
	}

	blocks, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	var problems []string

	for _, key := range slices.Sorted(maps.Keys(blocks)) {
		if reason := harnessBlockProblem(key, blocks[key]); reason != "" {
			problems = append(problems, FieldHarnessConfig+"."+key+reason)
		}
	}

	return problems
}

// harnessBlockProblem accepts one block: a key written as a slug, an object, a harness field on a
// variant naming one of the five and on nothing else, and each other field of its type. The reason
// it returns is relative to the block's own path, ": reason" for the block and ".field: reason" for
// a field, so the caller can put the path in front.
func harnessBlockProblem(key string, value any) string {
	if !harnessKeyRegexp.MatchString(key) {
		return ": the key must be lowercase letters and digits joined by hyphens, such as \"codex-direct\""
	}

	if current, former := harness.Renamed(key).Get(); former {
		return fmt.Sprintf(": %q is a former spelling of %q; key the block by the name the harness has now", key, current)
	}

	fields, ok := value.(map[string]any)
	if !ok {
		return ": must be an object"
	}

	named := harness.SkillsDir(key).IsPresent()
	runs, present := lookup(fields, FieldConfigHarness)

	switch {
	case named && present:
		return fmt.Sprintf(".%s: not allowed; the key %q already names the harness", FieldConfigHarness, key)
	case !named && !present:
		return fmt.Sprintf(".%s: missing; a key that is not a harness name says which harness runs it (%s)",
			FieldConfigHarness, quotedList(harness.All()))
	case !named:
		name, isText := runs.(string)
		if !isText {
			return fmt.Sprintf(".%s: must be a string", FieldConfigHarness)
		}

		if harness.SkillsDir(name).IsAbsent() {
			return fmt.Sprintf(".%s: %s", FieldConfigHarness, unknownValue(name, harness.All()))
		}
	}

	for _, field := range []string{FieldModelFlag, FieldProvider, FieldEnv} {
		if fieldValue, present := lookup(fields, field); present {
			if _, isText := fieldValue.(string); !isText {
				return fmt.Sprintf(".%s: must be a string", field)
			}
		}
	}

	if args, present := lookup(fields, FieldArgs); present {
		items, isList := args.([]any)
		if !isList {
			return fmt.Sprintf(".%s: must be an array of strings", FieldArgs)
		}

		for _, item := range items {
			if _, isText := item.(string); !isText {
				return fmt.Sprintf(".%s: must be an array of strings", FieldArgs)
			}
		}
	}

	return ""
}

// agentHarnessProblem checks every list item's harness against the document as a whole: a harness
// name, current, or a key of an accepted harnessConfig block. It runs after isAgents has accepted
// the list's shape, and after the harnessConfig pass, so a key it does not find is one the person
// has already been told about or one that is not there.
func agentHarnessProblem(doc Document) string {
	value, present := lookup(doc, FieldAgents)
	if !present {
		return ""
	}

	items, _ := value.([]any)
	keys := HarnessConfigKeys(doc)

	for i, item := range items {
		fields, _ := item.(map[string]any)

		for _, feature := range Features() {
			agents, _ := fields[feature].([]any)

			for j, agent := range agents {
				agentFields, _ := agent.(map[string]any)
				name, _ := agentFields[FieldAgentHarness].(string)

				if name == HarnessCurrent || harness.SkillsDir(name).IsPresent() || slices.Contains(keys, name) {
					continue
				}

				return fmt.Sprintf("[%d].%s[%d].%s: %s", i, feature, j, FieldAgentHarness, unknownAgentHarness(name, keys))
			}
		}
	}

	return ""
}

// unknownAgentHarness is how a list item's harness that is none of the three forms is reported: the
// value, the harnesses and current, and the harnessConfig keys the document has, when it has any.
func unknownAgentHarness(name string, keys []string) string {
	expected := quotedList(AgentHarnesses())

	if len(keys) == 0 {
		return fmt.Sprintf("unknown value %q (expected %s, or a %s key)", name, expected, FieldHarnessConfig)
	}

	return fmt.Sprintf("unknown value %q (expected %s, or a %s key: %s)", name, expected, FieldHarnessConfig, quotedList(keys))
}

func quotedList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}

	return strings.Join(quoted, ", ")
}
