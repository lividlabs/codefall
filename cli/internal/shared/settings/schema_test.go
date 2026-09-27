package settings

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
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

	if got, want := schemaList(t, review, "required"), RequiredReviewFields(); !slices.Equal(got, want) {
		t.Errorf("properties.%s.required = %q, want %q", BlockReview, got, want)
	}

	// The block is optional at the top level, so it must not appear in the schema's own required
	// list: settings written before it existed are still valid settings.
	if slices.Contains(schemaList(t, schema, "required"), BlockReview) {
		t.Errorf("required contains %q, want the review block to stay optional", BlockReview)
	}

	reviewProperties := schemaObject(t, review, "properties")

	if got := schemaText(t, schemaObject(t, schemaObject(t, reviewProperties, FieldReviewAgents), "items"), "type"); got != "string" {
		t.Errorf("properties.%s.properties.%s.items.type = %q, want string", BlockReview, FieldReviewAgents, got)
	}

	// The agents list: a name that is a slug, a harness that is one codefall can start or current,
	// and nothing required of the document as a whole, because absent means the default (ADR-009).
	agents := schemaObject(t, properties, FieldAgents)
	agentItems := schemaObject(t, agents, "items")

	if got, want := schemaList(t, agentItems, "required"), []string{FieldAgentName, FieldAgentHarness}; !slices.Equal(got, want) {
		t.Errorf("properties.%s.items.required = %q, want %q", FieldAgents, got, want)
	}

	agentProperties := schemaObject(t, agentItems, "properties")

	if got := schemaText(t, schemaObject(t, agentProperties, FieldAgentName), "pattern"); got != AgentNamePattern {
		t.Errorf("properties.%s.items.properties.%s.pattern = %q, want %q", FieldAgents, FieldAgentName, got, AgentNamePattern)
	}

	if got, want := schemaList(t, schemaObject(t, agentProperties, FieldAgentHarness), "enum"), AgentHarnesses(); !slices.Equal(got, want) {
		t.Errorf("properties.%s.items.properties.%s.enum = %q, want %q", FieldAgents, FieldAgentHarness, got, want)
	}

	if got := schemaText(t, schemaObject(t, agentProperties, FieldAgentModel), "type"); got != "string" {
		t.Errorf("properties.%s.items.properties.%s.type = %q, want string", FieldAgents, FieldAgentModel, got)
	}

	for _, field := range []string{FieldAgents, FieldAgentsByHarness} {
		if slices.Contains(schemaList(t, schema, "required"), field) {
			t.Errorf("required contains %q, want it to stay optional", field)
		}
	}

	// The override is keyed by exactly the harnesses codefall can set up: current is not a key,
	// because it is what the key resolves.
	byHarness := schemaObject(t, properties, FieldAgentsByHarness)
	byHarnessProperties := schemaObject(t, byHarness, "properties")

	if got, want := slices.Sorted(maps.Keys(byHarnessProperties)), harness.All(); !slices.Equal(got, want) {
		t.Errorf("properties.%s.properties keys = %q, want %q", FieldAgentsByHarness, got, want)
	}

	if extra, ok := byHarness["additionalProperties"].(bool); !ok || extra {
		t.Errorf("properties.%s.additionalProperties = %v, want false", FieldAgentsByHarness, byHarness["additionalProperties"])
	}

	for _, name := range harness.All() {
		if got := schemaText(t, schemaObject(t, schemaObject(t, byHarnessProperties, name), "items"), "type"); got != "string" {
			t.Errorf("properties.%s.properties.%s.items.type = %q, want string", FieldAgentsByHarness, name, got)
		}
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
