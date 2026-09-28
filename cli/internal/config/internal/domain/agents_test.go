package domain

import (
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

func TestInsertAt(t *testing.T) {
	order := []string{"subagent", "architect"}

	for _, tc := range []struct {
		name   string
		before mo.Option[string]
		after  mo.Option[string]
		want   int
	}{
		{name: "neither goes last", before: mo.None[string](), after: mo.None[string](), want: 2},
		{name: "before the first", before: mo.Some("subagent"), after: mo.None[string](), want: 0},
		{name: "after the first", before: mo.None[string](), after: mo.Some("subagent"), want: 1},
		{name: "after the last", before: mo.None[string](), after: mo.Some("architect"), want: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := InsertAt(order, tc.before, tc.after)
			if err != nil || got != tc.want {
				t.Errorf("InsertAt = %d, %v, want %d, nil", got, err, tc.want)
			}
		})
	}

	_, err := InsertAt(order, mo.Some("reviewer"), mo.None[string]())
	if err == nil || !strings.Contains(err.Error(), `no agent named "reviewer"`) ||
		!strings.Contains(err.Error(), "subagent, architect") {
		t.Errorf("InsertAt of an unknown name = %v, want it to name the name and the list", err)
	}
}

// Every problem with a proposed order is named at once.
func TestCheckOrder(t *testing.T) {
	current := []string{"subagent", "architect", "second"}

	if err := CheckOrder(current, []string{"second", "architect", "subagent"}); err != nil {
		t.Errorf("CheckOrder of a permutation = %v, want nil", err)
	}

	err := CheckOrder(current, []string{"second", "second", "reviewer"})
	if err == nil {
		t.Fatal("CheckOrder = nil, want the problems")
	}

	for _, want := range []string{"missing subagent, architect", "not in the list: reviewer", "named twice: second"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("CheckOrder error = %q, want it to say %q", err, want)
		}
	}
}

// Every order that still names an agent is found, in the order a person reads the file.
func TestReferrers(t *testing.T) {
	doc := settings.Document{
		settings.FieldAgents: []any{
			map[string]any{"name": "subagent", "harness": "current"},
			map[string]any{"name": "architect", "harness": "codex"},
		},
		settings.BlockReview:  map[string]any{"agents": []any{"architect", "subagent"}},
		settings.BlockConsult: map[string]any{"agents": []any{"subagent"}},
		settings.FieldAgentsByHarness: map[string]any{
			"codex":  []any{"subagent"},
			"claude": []any{"architect"},
		},
	}

	if got, want := Referrers(doc, "architect"), []string{"review.agents", "agentsByHarness.claude"}; !slices.Equal(got, want) {
		t.Errorf("Referrers(architect) = %q, want %q", got, want)
	}

	if got, want := Referrers(doc, "subagent"), []string{"review.agents", "consult.agents", "agentsByHarness.codex"}; !slices.Equal(got, want) {
		t.Errorf("Referrers(subagent) = %q, want %q", got, want)
	}

	if got := Referrers(doc, "second"); len(got) != 0 {
		t.Errorf("Referrers(second) = %q, want none", got)
	}
}
