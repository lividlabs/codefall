package settings

import (
	"reflect"
	"slices"
	"testing"

	"github.com/samber/mo"
)

// codexViaBedrock is a block under a harness's own name: Codex called through a provider, with a
// model flag, extra arguments, and a command that exports credentials first.
func codexViaBedrock() map[string]any {
	return map[string]any{
		"modelFlag": "--model",
		"provider":  "amazon-bedrock-runtime",
		"args":      []any{"-c", "model_reasoning_effort=high"},
		"env":       "aws configure export-credentials --format env",
	}
}

// codexDirect is a variant: the same binary, called without the provider.
func codexDirect() map[string]any {
	return map[string]any{
		"harness": "codex",
		"args":    []any{"-c", "model_reasoning_effort=high"},
	}
}

func withHarnessConfig(blocks map[string]any) Document {
	return with(complete(), FieldHarnessConfig, blocks)
}

func TestHarnessConfigs(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  Document
		want map[string]HarnessConfig
	}{
		{
			name: "absent means none",
			doc:  complete(),
			want: map[string]HarnessConfig{},
		},
		{
			name: "a harness-named block takes its harness from the key, a variant from its field",
			doc:  withHarnessConfig(map[string]any{"codex": codexViaBedrock(), "codex-direct": codexDirect()}),
			want: map[string]HarnessConfig{
				"codex": {
					Harness:   "codex",
					ModelFlag: mo.Some("--model"),
					Provider:  mo.Some("amazon-bedrock-runtime"),
					Env:       mo.Some("aws configure export-credentials --format env"),
					Args:      []string{"-c", "model_reasoning_effort=high"},
				},
				"codex-direct": {
					Harness:   "codex",
					ModelFlag: mo.None[string](),
					Provider:  mo.None[string](),
					Env:       mo.None[string](),
					Args:      []string{"-c", "model_reasoning_effort=high"},
				},
			},
		},
		{
			name: "an empty block under a harness name is that harness with nothing added",
			doc:  withHarnessConfig(map[string]any{"muse": map[string]any{}}),
			want: map[string]HarnessConfig{"muse": {
				Harness: "muse", ModelFlag: mo.None[string](), Provider: mo.None[string](), Env: mo.None[string](),
			}},
		},
		{
			name: "a block Validate refuses is left out, and the others stay",
			doc:  withHarnessConfig(map[string]any{"codex": codexViaBedrock(), "bedrock": map[string]any{"provider": "x"}}),
			want: map[string]HarnessConfig{"codex": {
				Harness:   "codex",
				ModelFlag: mo.Some("--model"),
				Provider:  mo.Some("amazon-bedrock-runtime"),
				Env:       mo.Some("aws configure export-credentials --format env"),
				Args:      []string{"-c", "model_reasoning_effort=high"},
			}},
		},
		{
			name: "an object that is not an object yields none",
			doc:  withHarnessConfig(nil),
			want: map[string]HarnessConfig{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := HarnessConfigs(tc.doc); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("HarnessConfigs() = %+v, want %+v", got, tc.want)
			}
		})
	}

	doc := withHarnessConfig(map[string]any{"codex": codexViaBedrock(), "codex-direct": codexDirect()})
	if got, want := HarnessConfigKeys(doc), []string{"codex", "codex-direct"}; !slices.Equal(got, want) {
		t.Errorf("HarnessConfigKeys() = %q, want %q", got, want)
	}
}

func TestResolveHarness(t *testing.T) {
	doc := withHarnessConfig(map[string]any{"codex": codexViaBedrock(), "codex-direct": codexDirect()})

	for _, tc := range []struct {
		key  string
		want mo.Option[string]
	}{
		{key: "codex", want: mo.Some("codex")},
		{key: "claude", want: mo.Some("claude")},
		{key: "current", want: mo.Some("current")},
		{key: "codex-direct", want: mo.Some("codex")},
		{key: "cursor", want: mo.None[string]()},
		{key: "claude-code", want: mo.None[string]()},
		{key: "default", want: mo.None[string]()},
	} {
		if got := ResolveHarness(doc, tc.key); got != tc.want {
			t.Errorf("ResolveHarness(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}

func TestValidateHarnessConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  Document
		want []string
	}{
		{
			name: "a harness-named block and a variant",
			doc:  withHarnessConfig(map[string]any{"codex": codexViaBedrock(), "codex-direct": codexDirect()}),
		},
		{
			name: "a variant with only its harness",
			doc:  withHarnessConfig(map[string]any{"codex-direct": map[string]any{"harness": "codex"}}),
		},
		{
			name: "the object is not an object",
			doc:  withHarnessConfig(nil),
		},
		{
			name: "the object is an array",
			doc:  with(complete(), FieldHarnessConfig, []any{}),
			want: []string{"harnessConfig: must be an object"},
		},
		{
			name: "a key that is not a slug",
			doc:  withHarnessConfig(map[string]any{"Codex Direct": codexDirect()}),
			want: []string{`harnessConfig.Codex Direct: the key must be lowercase letters and digits joined by hyphens, such as "codex-direct"`},
		},
		{
			name: "a key that is a former spelling of a harness",
			doc:  withHarnessConfig(map[string]any{"claude-code": map[string]any{"provider": "x"}}),
			want: []string{`harnessConfig.claude-code: "claude-code" is a former spelling of "claude"; key the block by the name the harness has now`},
		},
		{
			name: "a block that is not an object",
			doc:  withHarnessConfig(map[string]any{"codex": "bedrock"}),
			want: []string{"harnessConfig.codex: must be an object"},
		},
		{
			name: "a harness-named key carrying harness",
			doc:  withHarnessConfig(map[string]any{"codex": map[string]any{"harness": "codex"}}),
			want: []string{`harnessConfig.codex.harness: not allowed; the key "codex" already names the harness`},
		},
		{
			name: "a variant without harness",
			doc:  withHarnessConfig(map[string]any{"bedrock": map[string]any{"provider": "amazon-bedrock-runtime"}}),
			want: []string{`harnessConfig.bedrock.harness: missing; a key that is not a harness name says which harness runs it ` +
				`("agy", "claude", "codex", "muse", "opencode")`},
		},
		{
			name: "a variant on current",
			doc:  withHarnessConfig(map[string]any{"mine": map[string]any{"harness": "current"}}),
			want: []string{`harnessConfig.mine.harness: unknown value "current" (expected "agy", "claude", "codex", "muse", "opencode")`},
		},
		{
			name: "a variant on a former spelling",
			doc:  withHarnessConfig(map[string]any{"mine": map[string]any{"harness": "claude-code"}}),
			want: []string{`harnessConfig.mine.harness: unknown value "claude-code" (expected "agy", "claude", "codex", "muse", "opencode")`},
		},
		{
			name: "a variant whose harness is not a string",
			doc:  withHarnessConfig(map[string]any{"mine": map[string]any{"harness": 5.0}}),
			want: []string{"harnessConfig.mine.harness: must be a string"},
		},
		{
			name: "a model flag that is not a string",
			doc:  withHarnessConfig(map[string]any{"codex": map[string]any{"modelFlag": true}}),
			want: []string{"harnessConfig.codex.modelFlag: must be a string"},
		},
		{
			name: "a provider that is not a string",
			doc:  withHarnessConfig(map[string]any{"codex": map[string]any{"provider": []any{}}}),
			want: []string{"harnessConfig.codex.provider: must be a string"},
		},
		{
			name: "an env that is not a string",
			doc:  withHarnessConfig(map[string]any{"codex": map[string]any{"env": 1.0}}),
			want: []string{"harnessConfig.codex.env: must be a string"},
		},
		{
			name: "args that are not an array",
			doc:  withHarnessConfig(map[string]any{"codex": map[string]any{"args": "-c"}}),
			want: []string{"harnessConfig.codex.args: must be an array of strings"},
		},
		{
			name: "args holding something other than a string",
			doc:  withHarnessConfig(map[string]any{"codex": map[string]any{"args": []any{"-c", 2.0}}}),
			want: []string{"harnessConfig.codex.args: must be an array of strings"},
		},
		{
			name: "one problem per block, in key order",
			doc: withHarnessConfig(map[string]any{
				"muse":  map[string]any{"env": 1.0},
				"agy":   "bare",
				"mine":  map[string]any{},
				"codex": codexViaBedrock(),
			}),
			want: []string{
				"harnessConfig.agy: must be an object",
				`harnessConfig.mine.harness: missing; a key that is not a harness name says which harness runs it ` +
					`("agy", "claude", "codex", "muse", "opencode")`,
				"harnessConfig.muse.env: must be a string",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Validate(tc.doc); !slices.Equal(got, tc.want) {
				t.Errorf("Validate() = %q, want %q", got, tc.want)
			}
		})
	}
}

// A list item's harness may be a harnessConfig key, and only a key of a block Validate accepts.
func TestValidateAgentsAgainstHarnessConfig(t *testing.T) {
	list := func(runs string) []any {
		return []any{map[string]any{"activeAgent": "claude", "review": []any{map[string]any{"harness": runs}}}}
	}

	for _, tc := range []struct {
		name string
		doc  Document
		want []string
	}{
		{
			name: "a variant key",
			doc:  with(withHarnessConfig(map[string]any{"codex-direct": codexDirect()}), FieldAgents, list("codex-direct")),
		},
		{
			name: "a harness-named key is the harness, block or no block",
			doc:  with(withHarnessConfig(map[string]any{"codex": codexViaBedrock()}), FieldAgents, list("codex")),
		},
		{
			name: "a key the document does not hold, with keys it does",
			doc:  with(withHarnessConfig(map[string]any{"codex-direct": codexDirect()}), FieldAgents, list("bedrock")),
			want: []string{`agents: [0].review[0].harness: unknown value "bedrock" ` +
				`(expected "agy", "claude", "codex", "current", "muse", "opencode", or a harnessConfig key: "codex-direct")`},
		},
		{
			// The block is reported, and the key that names it is not accepted on the strength of a
			// block nothing can read.
			name: "a key whose block is refused",
			doc:  with(withHarnessConfig(map[string]any{"bedrock": map[string]any{"provider": "x"}}), FieldAgents, list("bedrock")),
			want: []string{
				`harnessConfig.bedrock.harness: missing; a key that is not a harness name says which harness runs it ` +
					`("agy", "claude", "codex", "muse", "opencode")`,
				`agents: [0].review[0].harness: unknown value "bedrock" ` +
					`(expected "agy", "claude", "codex", "current", "muse", "opencode", or a harnessConfig key)`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Validate(tc.doc); !slices.Equal(got, tc.want) {
				t.Errorf("Validate() = %q, want %q", got, tc.want)
			}
		})
	}

	// A list Validate refuses for its harness name reads as the default, like any other refusal.
	doc := with(withHarnessConfig(map[string]any{"codex-direct": codexDirect()}), FieldAgents, list("bedrock"))
	if got := Agents(doc); !reflect.DeepEqual(got, DefaultAgents()) {
		t.Errorf("Agents() with an unknown key = %+v, want the default", got)
	}

	doc = with(withHarnessConfig(map[string]any{"codex-direct": codexDirect()}), FieldAgents, list("codex-direct"))
	if got := Order(doc, "claude", FeatureReview); !reflect.DeepEqual(got, []Agent{{Harness: "codex-direct", Model: mo.None[string]()}}) {
		t.Errorf("Order() with a variant key = %+v, want the variant", got)
	}
}
