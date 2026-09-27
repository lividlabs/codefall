package settings

import (
	"slices"
	"testing"

	"github.com/samber/mo"
)

// twoAgents is a list that reaches for Codex first and this harness's own subagent after.
func twoAgents() []any {
	return []any{
		map[string]any{"name": "architect", "harness": "codex", "model": "gpt-5-codex"},
		map[string]any{"name": "subagent", "harness": "current"},
	}
}

func TestDefaultAgents(t *testing.T) {
	got := DefaultAgents()

	want := []Agent{{Name: DefaultAgentName, Harness: HarnessCurrent, Model: mo.None[string]()}}
	if !slices.Equal(got, want) {
		t.Errorf("DefaultAgents() = %+v, want %+v", got, want)
	}

	// The default is valid settings, or init would write a file doctor rejects.
	if problems := Validate(with(complete(), FieldAgents, []any{
		map[string]any{"name": DefaultAgentName, "harness": HarnessCurrent},
	})); len(problems) != 0 {
		t.Errorf("Validate(default agents) = %q, want none", problems)
	}
}

func TestAgentHarnesses(t *testing.T) {
	want := []string{"agy", "claude", "codex", "current", "muse", "opencode"}
	if got := AgentHarnesses(); !slices.Equal(got, want) {
		t.Errorf("AgentHarnesses() = %q, want %q", got, want)
	}
}

func TestAgents(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  Document
		want []Agent
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
			name: "the list as written, in order",
			doc:  with(complete(), FieldAgents, twoAgents()),
			want: []Agent{
				{Name: "architect", Harness: "codex", Model: mo.Some("gpt-5-codex")},
				{Name: "subagent", Harness: "current", Model: mo.None[string]()},
			},
		},
		{
			// The caller has already been told what is wrong with it by Validate.
			name: "a list Validate rejects reads as absent",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"name": "x", "harness": "cursor"}}),
			want: DefaultAgents(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Agents(tc.doc); !slices.Equal(got, tc.want) {
				t.Errorf("Agents() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestAgentNamesAndDescriptions(t *testing.T) {
	agents := Agents(with(complete(), FieldAgents, twoAgents()))

	if got, want := AgentNames(agents), []string{"architect", "subagent"}; !slices.Equal(got, want) {
		t.Errorf("AgentNames() = %q, want %q", got, want)
	}

	if got, want := DescribeAgents(agents), "architect (codex, gpt-5-codex), subagent (current)"; got != want {
		t.Errorf("DescribeAgents() = %q, want %q", got, want)
	}
}

func TestReviewAgents(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  Document
		want mo.Option[[]string]
	}{
		{
			name: "no review block",
			doc:  complete(),
			want: mo.None[[]string](),
		},
		{
			name: "a review block with no order of its own",
			doc:  with(complete(), BlockReview, map[string]any{"postToPullRequest": true}),
			want: mo.None[[]string](),
		},
		{
			name: "review's own order",
			doc: with(with(complete(), FieldAgents, twoAgents()), BlockReview,
				map[string]any{"postToPullRequest": true, "agents": []any{"subagent", "architect"}}),
			want: mo.Some([]string{"subagent", "architect"}),
		},
		{
			name: "an order that is not a list of names reads as none",
			doc:  with(complete(), BlockReview, map[string]any{"postToPullRequest": true, "agents": "subagent"}),
			want: mo.None[[]string](),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ReviewAgents(tc.doc)

			if got.IsPresent() != tc.want.IsPresent() || !slices.Equal(got.OrEmpty(), tc.want.OrEmpty()) {
				t.Errorf("ReviewAgents() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAgentsByHarness(t *testing.T) {
	doc := with(with(complete(), FieldAgents, twoAgents()), FieldAgentsByHarness, map[string]any{
		"claude": []any{"architect", "subagent"},
		"codex":  []any{"subagent"},
	})

	got := AgentsByHarness(doc)

	if len(got) != 2 {
		t.Fatalf("AgentsByHarness() = %v, want two harnesses", got)
	}

	if want := []string{"architect", "subagent"}; !slices.Equal(got["claude"], want) {
		t.Errorf("AgentsByHarness()[claude] = %q, want %q", got["claude"], want)
	}

	if want := []string{"subagent"}; !slices.Equal(got["codex"], want) {
		t.Errorf("AgentsByHarness()[codex] = %q, want %q", got["codex"], want)
	}

	if got := AgentsByHarness(complete()); len(got) != 0 {
		t.Errorf("AgentsByHarness(no override) = %v, want none", got)
	}
}

func TestHasCurrent(t *testing.T) {
	agents := Agents(with(complete(), FieldAgents, twoAgents()))

	if !HasCurrent(agents, []string{"architect", "subagent"}) {
		t.Error("HasCurrent(both) = false, want true")
	}

	if HasCurrent(agents, []string{"architect"}) {
		t.Error("HasCurrent(architect alone) = true, want false")
	}

	if HasCurrent(agents, nil) {
		t.Error("HasCurrent(empty order) = true, want false")
	}
}

func TestValidateAgents(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  Document
		want []string
	}{
		{
			name: "a complete list with review's order and a per-harness override",
			doc: with(with(with(complete(), FieldAgents, twoAgents()),
				BlockReview, map[string]any{"postToPullRequest": false, "agents": []any{"architect"}}),
				FieldAgentsByHarness, map[string]any{"codex": []any{"subagent"}}),
		},
		{
			name: "the list is not an array",
			doc:  with(complete(), FieldAgents, "subagent"),
			want: []string{"agents: must be an array of agents"},
		},
		{
			name: "an entry is not an object",
			doc:  with(complete(), FieldAgents, []any{"subagent"}),
			want: []string{"agents: [0]: must be an object"},
		},
		{
			name: "an entry has no name",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"harness": "current"}}),
			want: []string{"agents: [0].name: missing"},
		},
		{
			name: "a name is not a slug",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"name": "My Agent", "harness": "current"}}),
			want: []string{"agents: [0].name: must match " + AgentNamePattern},
		},
		{
			name: "a name is used twice",
			doc: with(complete(), FieldAgents, []any{
				map[string]any{"name": "subagent", "harness": "current"},
				map[string]any{"name": "subagent", "harness": "codex"},
			}),
			want: []string{`agents: names "subagent" twice`},
		},
		{
			name: "an entry has no harness",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"name": "subagent"}}),
			want: []string{"agents: [0].harness: missing"},
		},
		{
			name: "a harness codefall cannot start",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"name": "helper", "harness": "cursor"}}),
			want: []string{`agents: [0].harness: unknown value "cursor" ` +
				`(expected "agy", "claude", "codex", "current", "muse", "opencode")`},
		},
		{
			// The list is newer than the rename, so nothing checked in can carry a former spelling,
			// and one is refused rather than read as the name the harness has now.
			name: "a harness under its former spelling",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"name": "helper", "harness": "claude-code"}}),
			want: []string{`agents: [0].harness: unknown value "claude-code" ` +
				`(expected "agy", "claude", "codex", "current", "muse", "opencode")`},
		},
		{
			name: "a model that is not a string",
			doc:  with(complete(), FieldAgents, []any{map[string]any{"name": "helper", "harness": "codex", "model": 5.0}}),
			want: []string{"agents: [0].model: must be a string"},
		},
		{
			name: "review's order is not a list",
			doc:  with(complete(), BlockReview, map[string]any{"postToPullRequest": false, "agents": "subagent"}),
			want: []string{"review.agents: must be an array of agent names"},
		},
		{
			name: "review's order names an agent twice",
			doc:  with(complete(), BlockReview, map[string]any{"postToPullRequest": false, "agents": []any{"subagent", "subagent"}}),
			want: []string{`review.agents: names "subagent" twice`},
		},
		{
			// The default list defines subagent, so an order naming only that is fine without a list.
			name: "review's order over the default list",
			doc:  with(complete(), BlockReview, map[string]any{"postToPullRequest": false, "agents": []any{"subagent"}}),
		},
		{
			name: "review's order names an agent the list does not define",
			doc: with(with(complete(), FieldAgents, twoAgents()),
				BlockReview, map[string]any{"postToPullRequest": false, "agents": []any{"architect", "reviewer"}}),
			want: []string{`review.agents: names "reviewer", which agents does not define`},
		},
		{
			name: "the override is not an object",
			doc:  with(complete(), FieldAgentsByHarness, []any{"subagent"}),
			want: []string{"agentsByHarness: must be an object"},
		},
		{
			name: "the override is keyed by something that is not a harness",
			doc:  with(complete(), FieldAgentsByHarness, map[string]any{"current": []any{"subagent"}}),
			want: []string{`agentsByHarness.current: unknown value "current" ` +
				`(expected "agy", "claude", "codex", "muse", "opencode")`},
		},
		{
			name: "an override order is not a list",
			doc:  with(complete(), FieldAgentsByHarness, map[string]any{"codex": "subagent"}),
			want: []string{"agentsByHarness.codex: must be an array of agent names"},
		},
		{
			name: "an override order names an agent the list does not define",
			doc: with(with(complete(), FieldAgents, twoAgents()),
				FieldAgentsByHarness, map[string]any{"claude": []any{"architect", "reviewer"}, "codex": []any{"nobody"}}),
			want: []string{
				`agentsByHarness.claude: names "reviewer", which agents does not define`,
				`agentsByHarness.codex: names "nobody", which agents does not define`,
			},
		},
		{
			// An order checked against a list that was itself refused would say the same thing twice.
			name: "a rejected list suppresses the order problems",
			doc: with(with(complete(), FieldAgents, "nope"),
				BlockReview, map[string]any{"postToPullRequest": false, "agents": []any{"reviewer"}}),
			want: []string{"agents: must be an array of agents"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Validate(tc.doc)

			if len(got) == 0 && len(tc.want) == 0 {
				return
			}

			if !slices.Equal(got, tc.want) {
				t.Errorf("Validate() = %q, want %q", got, tc.want)
			}
		})
	}
}
