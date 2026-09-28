package settings

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
)

// schemas/settings.schema.json is the published definition of .codefall/settings.json; this package
// validates the same shape by hand, so no schema library ships in the binary. This test holds the
// two equal — without it they would drift apart the first time a field is added on one side. It is
// one test rather than one per component: initcmd writes the file and doctor reads it, and a file
// initcmd wrote that doctor rejected would be the worst possible bug in either.
//
// It lives here, beside the format it pins, which it could not do while the format lived in a
// domain/ package — depguard denies os in domain tests, and //go:embed cannot reach outside its own
// package directory, so a copy of the schema under domain/ would have been exactly the drift this
// test exists to catch. `go test` runs with the package directory as its working directory, so the
// relative path is stable.
const schemaPath = "../../../schemas/settings.schema.json"

func loadSchema(t *testing.T) map[string]any {
	t.Helper()

	data, err := os.ReadFile(filepath.FromSlash(schemaPath))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}

	return schema
}

func TestSchemaIdentity(t *testing.T) {
	schema := loadSchema(t)

	if got := schemaText(t, schema, "$id"); got != SchemaID {
		t.Errorf("$id = %q, want %q", got, SchemaID)
	}

	const draft = "https://json-schema.org/draft/2020-12/schema"
	if got := schemaText(t, schema, "$schema"); got != draft {
		t.Errorf("$schema = %q, want %q", got, draft)
	}
}

func TestSchemaMatchesTheFieldTables(t *testing.T) {
	schema := loadSchema(t)

	if got, want := schemaList(t, schema, "required"), RequiredFields(); !slices.Equal(got, want) {
		t.Errorf("required = %q, want %q", got, want)
	}

	properties := schemaObject(t, schema, "properties")

	if got := schemaNumber(t, schemaObject(t, properties, "version"), "const"); got != Version {
		t.Errorf("properties.version.const = %v, want %d", got, Version)
	}

	if got, want := schemaList(t, schemaObject(t, properties, "tracker"), "enum"), Trackers(); !slices.Equal(got, want) {
		t.Errorf("properties.tracker.enum = %q, want %q", got, want)
	}

	harnesses := schemaObject(t, properties, FieldHarnesses)

	// At least one, and only the harnesses codefall can set up — the same two rules the validator
	// applies, so a harness added to the shared module and not to the schema fails here.
	if got := schemaNumber(t, harnesses, "minItems"); got != 1 {
		t.Errorf("properties.%s.minItems = %v, want 1", FieldHarnesses, got)
	}

	if got, want := schemaList(t, schemaObject(t, harnesses, "items"), "enum"), harness.All(); !slices.Equal(got, want) {
		t.Errorf("properties.%s.items.enum = %q, want %q", FieldHarnesses, got, want)
	}

	if !schemaBool(t, harnesses, "uniqueItems") {
		t.Errorf("properties.%s.uniqueItems is false, want the validator's no-duplicates rule", FieldHarnesses)
	}

	beadsBlock := schemaObject(t, properties, TrackerBeads)

	if got, want := schemaRequiredList(t, beadsBlock, "required"), RequiredTrackerFields(TrackerBeads); !slices.Equal(got, want) {
		t.Errorf("properties.%s.required = %q, want %q", TrackerBeads, got, want)
	}

	block := schemaObject(t, properties, TrackerGitHub)

	if got, want := schemaList(t, block, "required"), RequiredTrackerFields(TrackerGitHub); !slices.Equal(got, want) {
		t.Errorf("properties.%s.required = %q, want %q", TrackerGitHub, got, want)
	}

	blockProperties := schemaObject(t, block, "properties")

	if got := schemaText(t, schemaObject(t, blockProperties, "issuesRepo"), "pattern"); got != RepoPattern {
		t.Errorf("properties.%s.properties.issuesRepo.pattern = %q, want %q", TrackerGitHub, got, RepoPattern)
	}

	if got := schemaNumber(t, schemaObject(t, blockProperties, "issuesProject"), "minimum"); got != 1 {
		t.Errorf("properties.%s.properties.project.minimum = %v, want 1", TrackerGitHub, got)
	}

	review := schemaObject(t, properties, BlockReview)

	// Nothing in the block is required: posting is off unless the project said so.
	if _, required := review["required"]; required {
		t.Errorf("properties.%s.required is set, want every field of the review block optional", BlockReview)
	}

	// The block is optional at the top level, so it must not appear in the schema's own required
	// list: settings written before it existed are still valid settings.
	if slices.Contains(schemaList(t, schema, "required"), BlockReview) {
		t.Errorf("required contains %q, want the review block to stay optional", BlockReview)
	}

	// The agents list is keyed by active agent: each entry names one, from the harnesses codefall
	// can set up plus default, and holds a review list and a consult list of agents on a harness it
	// can start or current, at least one each when present. Nothing is required of the document as a
	// whole, because absent means the default (ADR-009.2).
	agents := schemaObject(t, properties, FieldAgents)
	entry := schemaObject(t, agents, "items")

	if got, want := schemaList(t, entry, "required"), []string{FieldActiveAgent}; !slices.Equal(got, want) {
		t.Errorf("properties.%s.items.required = %q, want %q", FieldAgents, got, want)
	}

	entryProperties := schemaObject(t, entry, "properties")

	if got, want := schemaList(t, schemaObject(t, entryProperties, FieldActiveAgent), "enum"), ActiveAgents(); !slices.Equal(got, want) {
		t.Errorf("properties.%s.items.properties.%s.enum = %q, want %q", FieldAgents, FieldActiveAgent, got, want)
	}

	if got, want := slices.Sorted(maps.Keys(entryProperties)), slices.Sorted(slices.Values(append(Features(), FieldActiveAgent))); !slices.Equal(got, want) {
		t.Errorf("properties.%s.items.properties keys = %q, want %q", FieldAgents, got, want)
	}

	// Both lists hold the one agent shape, defined once under $defs.
	const agentRef = "#/$defs/agent"

	for _, feature := range Features() {
		list := schemaObject(t, entryProperties, feature)

		if got := schemaNumber(t, list, "minItems"); got != 1 {
			t.Errorf("properties.%s.items.properties.%s.minItems = %v, want 1", FieldAgents, feature, got)
		}

		if got := schemaText(t, schemaObject(t, list, "items"), "$ref"); got != agentRef {
			t.Errorf("properties.%s.items.properties.%s.items.$ref = %q, want %q", FieldAgents, feature, got, agentRef)
		}
	}

	agent := schemaObject(t, schemaObject(t, schema, "$defs"), "agent")

	if got, want := schemaList(t, agent, "required"), []string{FieldAgentHarness}; !slices.Equal(got, want) {
		t.Errorf("$defs.agent.required = %q, want %q", got, want)
	}

	agentProperties := schemaObject(t, agent, "properties")

	// An agent's harness is a harness name, current, or a harnessConfig key, and a key is any slug,
	// so the schema pins the shape rather than a closed list, and its description names the three
	// forms for whoever reads the schema instead of this module.
	agentHarness := schemaObject(t, agentProperties, FieldAgentHarness)

	if _, closed := agentHarness["enum"]; closed {
		t.Errorf("$defs.agent.properties.%s.enum is set, want none: a %s key is accepted too", FieldAgentHarness, FieldHarnessConfig)
	}

	if got := schemaText(t, agentHarness, "pattern"); got != HarnessKeyPattern {
		t.Errorf("$defs.agent.properties.%s.pattern = %q, want %q", FieldAgentHarness, got, HarnessKeyPattern)
	}

	description := schemaText(t, agentHarness, "description")
	for _, form := range append(harness.All(), HarnessCurrent, FieldHarnessConfig) {
		if !strings.Contains(description, form) {
			t.Errorf("$defs.agent.properties.%s.description = %q, want it to name %q", FieldAgentHarness, description, form)
		}
	}

	if got := schemaText(t, schemaObject(t, agentProperties, FieldAgentModel), "type"); got != "string" {
		t.Errorf("$defs.agent.properties.%s.type = %q, want string", FieldAgentModel, got)
	}

	// The harnessConfig object is keyed by slug, and each block carries the five fields, typed, with
	// the harness field closed to the five names: a variant names a binary, never current.
	harnessConfig := schemaObject(t, properties, FieldHarnessConfig)

	if got := schemaText(t, harnessConfig, "type"); got != "object" {
		t.Errorf("properties.%s.type = %q, want object", FieldHarnessConfig, got)
	}

	if schemaBool(t, harnessConfig, "additionalProperties") {
		t.Errorf("properties.%s.additionalProperties is true, want false: a key must match the slug pattern", FieldHarnessConfig)
	}

	patterns := schemaObject(t, harnessConfig, "patternProperties")

	if got, want := slices.Sorted(maps.Keys(patterns)), []string{HarnessKeyPattern}; !slices.Equal(got, want) {
		t.Fatalf("properties.%s.patternProperties keys = %q, want %q", FieldHarnessConfig, got, want)
	}

	configBlock := schemaObject(t, patterns, HarnessKeyPattern)

	if got := schemaText(t, configBlock, "type"); got != "object" {
		t.Errorf("properties.%s block type = %q, want object", FieldHarnessConfig, got)
	}

	if !schemaBool(t, configBlock, "additionalProperties") {
		t.Errorf("properties.%s block additionalProperties is false, want true", FieldHarnessConfig)
	}

	blockFields := schemaObject(t, configBlock, "properties")

	if got, want := slices.Sorted(maps.Keys(blockFields)), slices.Sorted(slices.Values(HarnessConfigFields())); !slices.Equal(got, want) {
		t.Errorf("properties.%s block properties keys = %q, want %q", FieldHarnessConfig, got, want)
	}

	if got, want := schemaList(t, schemaObject(t, blockFields, FieldConfigHarness), "enum"), harness.All(); !slices.Equal(got, want) {
		t.Errorf("properties.%s block %s.enum = %q, want %q", FieldHarnessConfig, FieldConfigHarness, got, want)
	}

	for _, field := range []string{FieldConfigHarness, FieldModelFlag, FieldProvider, FieldEnv} {
		if got := schemaText(t, schemaObject(t, blockFields, field), "type"); got != "string" {
			t.Errorf("properties.%s block %s.type = %q, want string", FieldHarnessConfig, field, got)
		}
	}

	args := schemaObject(t, blockFields, FieldArgs)

	if got := schemaText(t, args, "type"); got != "array" {
		t.Errorf("properties.%s block %s.type = %q, want array", FieldHarnessConfig, FieldArgs, got)
	}

	if got := schemaText(t, schemaObject(t, args, "items"), "type"); got != "string" {
		t.Errorf("properties.%s block %s.items.type = %q, want string", FieldHarnessConfig, FieldArgs, got)
	}

	if slices.Contains(schemaList(t, schema, "required"), FieldHarnessConfig) {
		t.Errorf("required contains %q, want it to stay optional", FieldHarnessConfig)
	}

	if slices.Contains(schemaList(t, schema, "required"), FieldAgents) {
		t.Errorf("required contains %q, want it to stay optional", FieldAgents)
	}

	// Which agents review is in the agents list, by active agent, and nowhere else.
	for _, retired := range []string{"agentsByHarness", "consult"} {
		if _, has := properties[retired]; has {
			t.Errorf("properties.%s is present, want it gone: the agents list holds it now", retired)
		}
	}

	if _, has := schemaObject(t, review, "properties")["agents"]; has {
		t.Errorf("properties.%s.properties.agents is present, want it gone: the agents list holds it now", BlockReview)
	}

	local := schemaObject(t, properties, BlockLocal)

	if got, want := schemaList(t, local, "required"), RequiredLocalFields(); !slices.Equal(got, want) {
		t.Errorf("properties.%s.required = %q, want %q", BlockLocal, got, want)
	}

	localProperties := schemaObject(t, local, "properties")

	for _, field := range RequiredLocalFields() {
		if got := schemaText(t, schemaObject(t, localProperties, field), "type"); got != "string" {
			t.Errorf("properties.%s.properties.%s.type = %q, want string", BlockLocal, field, got)
		}
	}

	if slices.Contains(schemaList(t, schema, "required"), BlockLocal) {
		t.Errorf("required contains %q, want the local block to stay optional", BlockLocal)
	}

	test := schemaObject(t, properties, BlockTest)

	if got, want := schemaList(t, test, "required"), RequiredTestFields(); !slices.Equal(got, want) {
		t.Errorf("properties.%s.required = %q, want %q", BlockTest, got, want)
	}

	testProperties := schemaObject(t, test, "properties")

	// The directory's shape and the runners' names are the two rules the validator applies, so a
	// pattern loosened on one side or a runner added to the shared module alone fails here.
	if got := schemaText(t, schemaObject(t, testProperties, FieldTestDir), "pattern"); got != TestDirPattern {
		t.Errorf("properties.%s.properties.%s.pattern = %q, want %q", BlockTest, FieldTestDir, got, TestDirPattern)
	}

	runners := schemaObject(t, testProperties, FieldTestRunners)

	if got, want := schemaList(t, schemaObject(t, runners, "items"), "enum"), TestRunners(); !slices.Equal(got, want) {
		t.Errorf("properties.%s.properties.%s.items.enum = %q, want %q", BlockTest, FieldTestRunners, got, want)
	}

	if !schemaBool(t, runners, "uniqueItems") {
		t.Errorf("properties.%s.properties.%s.uniqueItems is false, want the validator's no-duplicates rule",
			BlockTest, FieldTestRunners)
	}

	if slices.Contains(schemaList(t, schema, "required"), BlockTest) {
		t.Errorf("required contains %q, want the test block to stay optional", BlockTest)
	}
}

// One oneOf branch per tracker, each pinning "tracker" to itself, requiring its own block, and
// forbidding every other tracker's block — which is what makes "exactly one tracker block" hold in
// the schema as well as in the validator. With one tracker the forbidding loop is empty.
func TestSchemaHasOneBranchPerTracker(t *testing.T) {
	schema := loadSchema(t)
	trackers := Trackers()

	branches, ok := schema["oneOf"].([]any)
	if !ok {
		t.Fatalf("oneOf is %T, want a list", schema["oneOf"])
	}

	if len(branches) != len(trackers) {
		t.Fatalf("oneOf has %d branches, want %d (one per tracker)", len(branches), len(trackers))
	}

	for i, tracker := range trackers {
		branch, ok := branches[i].(map[string]any)
		if !ok {
			t.Fatalf("oneOf[%d] is %T, want an object", i, branches[i])
		}

		properties := schemaObject(t, branch, "properties")

		if got := schemaText(t, schemaObject(t, properties, "tracker"), "const"); got != tracker {
			t.Errorf("oneOf[%d].properties.tracker.const = %q, want %q", i, got, tracker)
		}

		required := schemaList(t, branch, "required")
		for _, want := range []string{"tracker", tracker} {
			if !slices.Contains(required, want) {
				t.Errorf("oneOf[%d].required = %q, want it to contain %q", i, required, want)
			}
		}

		for _, other := range trackers {
			if other == tracker {
				continue
			}

			if forbidden, ok := properties[other].(bool); !ok || forbidden {
				t.Errorf("oneOf[%d].properties.%s = %v, want false", i, other, properties[other])
			}
		}
	}
}

func schemaObject(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("%q is %T, want an object", key, parent[key])
	}

	return value
}

func schemaText(t *testing.T, parent map[string]any, key string) string {
	t.Helper()

	value, ok := parent[key].(string)
	if !ok {
		t.Fatalf("%q is %T, want a string", key, parent[key])
	}

	return value
}

func schemaBool(t *testing.T, parent map[string]any, key string) bool {
	t.Helper()

	value, ok := parent[key].(bool)
	if !ok {
		t.Fatalf("%q is %T, want true or false", key, parent[key])
	}

	return value
}

func schemaNumber(t *testing.T, parent map[string]any, key string) int {
	t.Helper()

	value, ok := parent[key].(float64)
	if !ok {
		t.Fatalf("%q is %T, want a number", key, parent[key])
	}

	return int(value)
}

func schemaList(t *testing.T, parent map[string]any, key string) []string {
	t.Helper()

	raw, ok := parent[key].([]any)
	if !ok {
		t.Fatalf("%q is %T, want a list", key, parent[key])
	}

	items := make([]string, 0, len(raw))

	for i, item := range raw {
		text, ok := item.(string)
		if !ok {
			t.Fatalf("%q[%d] is %T, want a string", key, i, item)
		}

		items = append(items, text)
	}

	return items
}

// schemaRequiredList behaves like schemaList, but a block that omits "required" entirely — rather
// than writing an empty array — is treated as requiring nothing. The beads block has no fields
// today, so it omits the key; this keeps that block's required-fields assertion pinned to the field
// table without forcing an empty "required": [] into the schema for a tracker that carries no
// fields.
func schemaRequiredList(t *testing.T, parent map[string]any, key string) []string {
	t.Helper()

	if _, present := parent[key]; !present {
		return []string{}
	}

	return schemaList(t, parent, key)
}
