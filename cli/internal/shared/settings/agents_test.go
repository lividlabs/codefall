package settings

import (
	"reflect"
	"slices"
	"testing"

	"github.com/samber/mo"
)

// museEntry is an entry for a session running in Muse: Claude reviews, then Codex on a chosen
// model, and consult is left to the default entry.
func museEntry() map[string]any {
	return map[string]any{
		"activeAgent": "muse",
		"review": []any{
			map[string]any{"harness": "claude"},
			map[string]any{"harness": "codex", "model": "gpt-5-codex"},
		},
	}
}

// defaultEntry is the default entry written the way init writes it.
func defaultEntry() map[string]any {
	return map[string]any{
		"activeAgent": "default",
		"review":      []any{map[string]any{"harness": "current"}},
		"consult":     []any{map[string]any{"harness": "current"}},
	}
}

func TestDefaultAgents(t *testing.T) {
	got := DefaultAgents()

	want := []Entry{{
		ActiveAgent: ActiveDefault,
		Review:      mo.Some([]Agent{{Harness: HarnessCurrent, Model: mo.None[string]()}}),
		Consult:     mo.Some([]Agent{{Harness: HarnessCurrent, Model: mo.None[string]()}}),
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DefaultAgents() = %+v, want %+v", got, want)
	}

	// The default is valid settings, or init would write a file doctor rejects.
	if problems := Validate(with(complete(), FieldAgents, []any{defaultEntry()})); len(problems) != 0 {
		t.Errorf("Validate(default agents) = %q, want none", problems)
	}
}

func TestActiveAgentsAndAgentHarnesses(t *testing.T) {
	if got, want := ActiveAgents(), []string{"agy", "claude", "codex", "default", "muse", "opencode"}; !slices.Equal(got, want) {
		t.Errorf("ActiveAgents() = %q, want %q", got, want)
	}

	if got, want := AgentHarnesses(), []string{"agy", "claude", "codex", "current", "muse", "opencode"}; !slices.Equal(got, want) {
		t.Errorf("AgentHarnesses() = %q, want %q", got, want)
	}
}

func TestParseActiveAgent(t *testing.T) {
	for _, tc := range []struct {
		in, want string
		wantErr  bool
	}{
		{in: "default", want: "default"},
		{in: "muse", want: "muse"},
		{in: "claude-code", want: "claude"},
		{in: "current", wantErr: true},
		{in: "cursor", wantErr: true},
	} {
		got, err := ParseActiveAgent(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseActiveAgent(%q) error = %v, wantErr %v", tc.in, err, tc.wantErr)
		}

		if got != tc.want {
			t.Errorf("ParseActiveAgent(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseAgent(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    Agent
		wantErr string
	}{
		{in: "codex", want: Agent{Harness: "codex", Model: mo.None[string]()}},
		{in: "codex:gpt-5-codex", want: Agent{Harness: "codex", Model: mo.Some("gpt-5-codex")}},
		{in: "current", want: Agent{Harness: "current", Model: mo.None[string]()}},
		{in: "claude-code:opus", want: Agent{Harness: "claude", Model: mo.Some("opus")}},
		{in: "codex:", wantErr: `agent "codex:" names an empty model`},
		{in: "current:", wantErr: `agent "current:" names an empty model`},
		// A slug that is not a harness may be a harnessConfig key; the document decides.
		{in: "codex-direct:gpt-6-astra", want: Agent{Harness: "codex-direct", Model: mo.Some("gpt-6-astra")}},
		{in: "cursor", want: Agent{Harness: "cursor", Model: mo.None[string]()}},
		{in: "Codex", wantErr: `agent "Codex": harness "Codex" is not a harness (agy, claude, codex, muse, opencode), ` +
			`current, or a harnessConfig key, which is lowercase letters and digits joined by hyphens`},
		{in: "", wantErr: `agent "": harness "" is not a harness (agy, claude, codex, muse, opencode), ` +
			`current, or a harnessConfig key, which is lowercase letters and digits joined by hyphens`},
	} {
		got, err := ParseAgent(tc.in)

		if tc.wantErr != "" {
			if err == nil || err.Error() != tc.wantErr {
				t.Errorf("ParseAgent(%q) error = %v, want %q", tc.in, err, tc.wantErr)
			}

			continue
		}

		if err != nil {
			t.Errorf("ParseAgent(%q) error = %v", tc.in, err)
		}

		if got != tc.want {
			t.Errorf("ParseAgent(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestAgents(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  Document
		want []Entry
	}{
		{
			name: "absent means the default",
			doc:  complete(),
			want: DefaultAgents(),
		},
		{
			name: "an empty list means the default too",
			doc:  with(complete(), FieldAgents, []any{}),
			want: DefaultAgents(),
		},
		{
			name: "the entries as written, in order, with a list left out read as absent",
			doc:  with(complete(), FieldAgents, []any{defaultEntry(), museEntry()}),
			want: []Entry{
				DefaultAgents()[0],
				{
					ActiveAgent: "muse",
					Review: mo.Some([]Agent{
						{Harness: "claude", Model: mo.None[string]()},
						{Harness: "codex", Model: mo.Some("gpt-5-codex")},
					}),
					Consult: mo.None[[]Agent](),
				},
			},
		},
		{
			name: "a list Validate refuses reads as the default",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "cursor"}}),
			want: DefaultAgents(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Agents(tc.doc); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Agents() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestOrderResolvesTheActiveAgentThenTheDefaultThenCurrent(t *testing.T) {
	doc := with(complete(), FieldAgents, []any{
		map[string]any{
			"activeAgent": "default",
			"consult":     []any{map[string]any{"harness": "codex"}, map[string]any{"harness": "current"}},
		},
		museEntry(),
	})

	claude := Agent{Harness: "claude", Model: mo.None[string]()}
	codex := Agent{Harness: "codex", Model: mo.None[string]()}

	for _, tc := range []struct {
		active, feature string
		want            []Agent
	}{
		// Muse has its own review list.
		{"muse", FeatureReview, []Agent{claude, {Harness: "codex", Model: mo.Some("gpt-5-codex")}}},
		// Muse has no consult list, so the default entry's applies.
		{"muse", FeatureConsult, []Agent{codex, CurrentAgent()}},
		// Nobody wrote a review list for default, so current alone.
		{"claude", FeatureReview, []Agent{CurrentAgent()}},
		{"claude", FeatureConsult, []Agent{codex, CurrentAgent()}},
		{"default", FeatureReview, []Agent{CurrentAgent()}},
	} {
		if got := Order(doc, tc.active, tc.feature); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Order(%s, %s) = %+v, want %+v", tc.active, tc.feature, got, tc.want)
		}
	}

	// Settings that carry no list at all resolve to the default entry's lists.
	if got := Order(complete(), "codex", FeatureReview); !reflect.DeepEqual(got, []Agent{CurrentAgent()}) {
		t.Errorf("Order(no agents) = %+v, want current alone", got)
	}
}

func TestEntryFor(t *testing.T) {
	doc := with(complete(), FieldAgents, []any{defaultEntry(), museEntry()})

	if entry, ok := EntryFor(doc, "muse").Get(); !ok || entry.ActiveAgent != "muse" {
		t.Errorf("EntryFor(muse) = %+v, %v; want the muse entry", entry, ok)
	}

	if EntryFor(doc, "codex").IsPresent() {
		t.Error("EntryFor(codex) is present, want none")
	}
}

func TestHasCurrentAndDescriptions(t *testing.T) {
	agents := Order(with(complete(), FieldAgents, []any{museEntry()}), "muse", FeatureReview)

	if HasCurrent(agents) {
		t.Error("HasCurrent(claude, codex) = true, want false")
	}

	if !HasCurrent([]Agent{CurrentAgent()}) {
		t.Error("HasCurrent(current) = false, want true")
	}

	if got, want := DescribeAgents(agents), "claude, codex:gpt-5-codex"; got != want {
		t.Errorf("DescribeAgents() = %q, want %q", got, want)
	}

	if got, want := ActiveAgentNames(Agents(with(complete(), FieldAgents, []any{defaultEntry(), museEntry()}))),
		[]string{"default", "muse"}; !slices.Equal(got, want) {
		t.Errorf("ActiveAgentNames() = %q, want %q", got, want)
	}
}

func TestValidateAgents(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  Document
		want []string
	}{
		{
			name: "a default entry and one for muse",
			doc:  with(complete(), FieldAgents, []any{defaultEntry(), museEntry()}),
		},
		{
			name: "the list is not an array",
			doc:  with(complete(), FieldAgents, "current"),
			want: []string{"agents: must be an array of entries, one per active agent"},
		},
		{
			name: "an entry is not an object",
			doc:  with(complete(), FieldAgents, []any{"current"}),
			want: []string{"agents: [0]: must be an object"},
		},
		{
			name: "an entry names no active agent",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"review": []any{map[string]any{"harness": "current"}}}}),
			want: []string{"agents: [0].activeAgent: missing"},
		},
		{
			name: "an active agent codefall does not know",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "cursor"}}),
			want: []string{`agents: [0].activeAgent: unknown value "cursor" ` +
				`(expected "agy", "claude", "codex", "default", "muse", "opencode")`},
		},
		{
			// current is what a list resolves to, never a harness a session runs in.
			name: "current as an active agent",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "current"}}),
			want: []string{`agents: [0].activeAgent: unknown value "current" ` +
				`(expected "agy", "claude", "codex", "default", "muse", "opencode")`},
		},
		{
			name: "an active agent named twice",
			doc:  with(complete(), FieldAgents, []any{defaultEntry(), defaultEntry()}),
			want: []string{`agents: names activeAgent "default" twice`},
		},
		{
			name: "a list is not an array",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "muse", "review": "claude"}}),
			want: []string{"agents: [0].review: must be an array of agents"},
		},
		{
			name: "an empty list",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "muse", "consult": []any{}}}),
			want: []string{"agents: [0].consult: must name at least one agent; leave the key out to use the default entry's list"},
		},
		{
			name: "an agent is not an object",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "muse", "review": []any{"claude"}}}),
			want: []string{"agents: [0].review[0]: must be an object"},
		},
		{
			name: "an agent has no harness",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "muse", "review": []any{map[string]any{"model": "x"}}}}),
			want: []string{"agents: [0].review[0].harness: missing"},
		},
		{
			name: "a harness codefall cannot start",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "muse", "review": []any{map[string]any{"harness": "cursor"}}}}),
			want: []string{`agents: [0].review[0].harness: unknown value "cursor" ` +
				`(expected "agy", "claude", "codex", "current", "muse", "opencode", or a harnessConfig key)`},
		},
		{
			name: "a harness that is not a string",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "muse", "review": []any{map[string]any{"harness": 5.0}}}}),
			want: []string{"agents: [0].review[0].harness: must be a string"},
		},
		{
			// The list is newer than the rename, so nothing checked in can carry a former spelling,
			// and one is refused rather than read as the name the harness has now.
			name: "a harness under its former spelling",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "muse", "review": []any{map[string]any{"harness": "claude-code"}}}}),
			want: []string{`agents: [0].review[0].harness: unknown value "claude-code" ` +
				`(expected "agy", "claude", "codex", "current", "muse", "opencode", or a harnessConfig key)`},
		},
		{
			name: "a model that is not a string",
			doc: with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "muse",
				"consult": []any{map[string]any{"harness": "codex", "model": 5.0}}}}),
			want: []string{"agents: [0].consult[0].model: must be a string"},
		},
		{
			// An entry with neither list is allowed: it says nothing, and the default applies.
			name: "an entry with no lists",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"activeAgent": "codex"}}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Validate(tc.doc); !slices.Equal(got, tc.want) {
				t.Errorf("Validate() = %q, want %q", got, tc.want)
			}
		})
	}
}
