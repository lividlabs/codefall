package application

import (
	"errors"
	"strings"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
)

// orderSettings defines two agents and carries a review block without an order of its own, and no
// consult block or per-harness override, so a write has each shape to start from.
const orderSettings = `{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "agents": [
    {"name": "subagent", "harness": "current"},
    {"name": "architect", "harness": "codex"}
  ],
  "review": {
    "postToPullRequest": false
  },
  "note": "kept <as> written"
}
`

// orderedSettings carries all three orders, each laid out differently: review's across lines,
// consult's on one line inside a block that spans lines, and the per-harness override on one line.
const orderedSettings = `{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "agents": [
    {"name": "subagent", "harness": "current"},
    {"name": "architect", "harness": "codex"}
  ],
  "review": {
    "agents": [
      "subagent"
    ],
    "postToPullRequest": false
  },
  "consult": {
    "agents": ["subagent"]
  },
  "agentsByHarness": {"codex": ["subagent"]},
  "note": "kept <as> written"
}
`

// Setting an order changes only that order's text: a value already there is replaced in the layout
// it had, an order a block lacks is added in the block's own layout, and a block that is not there
// is made for it. Every other byte of the file stays as it was.
func TestSetOrder(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings string
		order    Order
		names    []string
		detail   string
		old, new string
	}{
		{
			name:     "review, into a block that has no order",
			settings: orderSettings,
			order:    ReviewOrder(),
			names:    []string{"architect", "subagent"},
			detail:   "set review.agents in .codefall/settings.json: architect, subagent",
			old: `    "postToPullRequest": false
  },`,
			new: `    "postToPullRequest": false,
    "agents": ["architect", "subagent"]
  },`,
		},
		{
			name:     "review, replacing an order that spans lines",
			settings: orderedSettings,
			order:    ReviewOrder(),
			names:    []string{"architect", "subagent"},
			detail:   "set review.agents in .codefall/settings.json: architect, subagent",
			old: `    "agents": [
      "subagent"
    ],`,
			new: `    "agents": [
      "architect",
      "subagent"
    ],`,
		},
		{
			// Nothing in the review block is required, so an order is enough to make one; posting to a
			// pull request stays off until the project says otherwise.
			name: "review, making the block",
			settings: strings.Replace(orderSettings, `  "review": {
    "postToPullRequest": false
  },
`, "", 1),
			order:  ReviewOrder(),
			names:  []string{"subagent"},
			detail: "set review.agents in .codefall/settings.json: subagent",
			old: `  "note": "kept <as> written"
}`,
			new: `  "note": "kept <as> written",
  "review": {
    "agents": ["subagent"]
  }
}`,
		},
		{
			name:     "consult, making the block",
			settings: orderSettings,
			order:    ConsultOrder(),
			names:    []string{"architect"},
			detail:   "set consult.agents in .codefall/settings.json: architect",
			old: `  "note": "kept <as> written"
}`,
			new: `  "note": "kept <as> written",
  "consult": {
    "agents": ["architect"]
  }
}`,
		},
		{
			name:     "consult, replacing an order on one line",
			settings: orderedSettings,
			order:    ConsultOrder(),
			names:    []string{"architect", "subagent"},
			detail:   "set consult.agents in .codefall/settings.json: architect, subagent",
			old:      `    "agents": ["subagent"]`,
			new:      `    "agents": ["architect", "subagent"]`,
		},
		{
			name: "consult, into an empty block",
			settings: strings.Replace(orderSettings, `  "note"`, `  "consult": {},
  "note"`, 1),
			order:  ConsultOrder(),
			names:  []string{"architect"},
			detail: "set consult.agents in .codefall/settings.json: architect",
			old:    `  "consult": {},`,
			new: `  "consult": {
    "agents": ["architect"]
  },`,
		},
		{
			name:     "a harness, making the override",
			settings: orderSettings,
			order:    HarnessOrder("claude"),
			names:    []string{"architect", "subagent"},
			detail:   "set agentsByHarness.claude in .codefall/settings.json: architect, subagent",
			old: `  "note": "kept <as> written"
}`,
			new: `  "note": "kept <as> written",
  "agentsByHarness": {
    "claude": ["architect", "subagent"]
  }
}`,
		},
		{
			name:     "a harness, added to an override on one line",
			settings: orderedSettings,
			order:    HarnessOrder("claude"),
			names:    []string{"architect"},
			detail:   "set agentsByHarness.claude in .codefall/settings.json: architect",
			old:      `  "agentsByHarness": {"codex": ["subagent"]},`,
			new:      `  "agentsByHarness": {"codex": ["subagent"], "claude": ["architect"]},`,
		},
		{
			name:     "a harness, replacing its order",
			settings: orderedSettings,
			order:    HarnessOrder("codex"),
			names:    []string{"subagent", "architect"},
			detail:   "set agentsByHarness.codex in .codefall/settings.json: subagent, architect",
			old:      `  "agentsByHarness": {"codex": ["subagent"]},`,
			new:      `  "agentsByHarness": {"codex": ["subagent", "architect"]},`,
		},
		{
			// A harness's former spelling is accepted, as it is everywhere, and written as the name
			// it has now, which is the only key the settings accept.
			name:     "a harness by its former spelling",
			settings: orderSettings,
			order:    HarnessOrder("claude-code"),
			names:    []string{"subagent"},
			detail:   "set agentsByHarness.claude in .codefall/settings.json: subagent",
			old: `  "note": "kept <as> written"
}`,
			new: `  "note": "kept <as> written",
  "agentsByHarness": {
    "claude": ["subagent"]
  }
}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := project(tc.settings)

			write, err := NewConfig(files).SetOrder(workingDir, tc.order, tc.names)
			if err != nil {
				t.Fatalf("SetOrder: %v", err)
			}

			if want := domain.Changed(tc.detail); write != want {
				t.Errorf("write = %+v, want %+v", write, want)
			}

			if !strings.Contains(tc.settings, tc.old) {
				t.Fatalf("the fixture does not hold %q", tc.old)
			}

			want := strings.Replace(tc.settings, tc.old, tc.new, 1)
			if got := string(files.files[settingsFull]); got != want {
				t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// Clearing an order removes its key and the comma that joined it, and nothing else. The consult block
// stays when it is left empty, which the schema allows; the per-harness override goes with its last
// entry, because it means nothing without one.
func TestClearOrder(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings string
		order    Order
		detail   string
		old, new string
	}{
		{
			name:     "review, the first key in its block",
			settings: orderedSettings,
			order:    ReviewOrder(),
			detail:   "removed review.agents from .codefall/settings.json",
			old: `  "review": {
    "agents": [
      "subagent"
    ],
    "postToPullRequest": false
  },`,
			new: `  "review": {
    "postToPullRequest": false
  },`,
		},
		{
			name:     "review, the last key in its block",
			settings: strings.Replace(orderSettings, `"postToPullRequest": false`, `"postToPullRequest": false, "agents": ["architect"]`, 1),
			order:    ReviewOrder(),
			detail:   "removed review.agents from .codefall/settings.json",
			old:      `"postToPullRequest": false, "agents": ["architect"]`,
			new:      `"postToPullRequest": false`,
		},
		{
			name:     "consult, leaving the block empty",
			settings: orderedSettings,
			order:    ConsultOrder(),
			detail:   "removed consult.agents from .codefall/settings.json",
			old: `  "consult": {
    "agents": ["subagent"]
  },`,
			new: `  "consult": {},`,
		},
		{
			name:     "a harness, the override's last entry",
			settings: orderedSettings,
			order:    HarnessOrder("codex"),
			detail:   "removed agentsByHarness.codex from .codefall/settings.json, and the empty agentsByHarness with it",
			old: `  "agentsByHarness": {"codex": ["subagent"]},
`,
			new: ``,
		},
		{
			name: "a harness, leaving another",
			settings: strings.Replace(orderedSettings, `{"codex": ["subagent"]}`, `{
    "claude": ["architect"],
    "codex": ["subagent"]
  }`, 1),
			order:  HarnessOrder("claude"),
			detail: "removed agentsByHarness.claude from .codefall/settings.json",
			old: `{
    "claude": ["architect"],
    "codex": ["subagent"]
  }`,
			new: `{
    "codex": ["subagent"]
  }`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := project(tc.settings)

			write, err := NewConfig(files).ClearOrder(workingDir, tc.order)
			if err != nil {
				t.Fatalf("ClearOrder: %v", err)
			}

			if want := domain.Changed(tc.detail); write != want {
				t.Errorf("write = %+v, want %+v", write, want)
			}

			if !strings.Contains(tc.settings, tc.old) {
				t.Fatalf("the fixture does not hold %q", tc.old)
			}

			want := strings.Replace(tc.settings, tc.old, tc.new, 1)
			if got := string(files.files[settingsFull]); got != want {
				t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// Asking for what the file already says writes nothing and says so, so a script can run the same
// command twice.
func TestOrderWritesThatChangeNothing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		write func(*Config) (domain.Write, error)
		want  string
	}{
		{
			name: "review set to the order it has",
			write: func(c *Config) (domain.Write, error) {
				return c.SetOrder(workingDir, ReviewOrder(), []string{"subagent"})
			},
			want: ".codefall/settings.json already sets review.agents to subagent",
		},
		{
			name: "consult set to the order it has",
			write: func(c *Config) (domain.Write, error) {
				return c.SetOrder(workingDir, ConsultOrder(), []string{"subagent"})
			},
			want: ".codefall/settings.json already sets consult.agents to subagent",
		},
		{
			name: "a harness set to the order it has",
			write: func(c *Config) (domain.Write, error) {
				return c.SetOrder(workingDir, HarnessOrder("codex"), []string{"subagent"})
			},
			want: ".codefall/settings.json already sets agentsByHarness.codex to subagent",
		},
		{
			name:  "a harness with no order cleared",
			write: func(c *Config) (domain.Write, error) { return c.ClearOrder(workingDir, HarnessOrder("claude")) },
			want:  ".codefall/settings.json has no agentsByHarness.claude to clear",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := project(orderedSettings)

			write, err := tc.write(NewConfig(files))
			if err != nil || write != domain.Unchanged(tc.want) {
				t.Errorf("write = %+v, %v, want unchanged saying %q", write, err, tc.want)
			}

			if len(files.writes) != 0 {
				t.Errorf("wrote %q, want nothing", files.writes)
			}
		})
	}

	files := project(orderSettings)

	for _, order := range []Order{ReviewOrder(), ConsultOrder()} {
		write, err := NewConfig(files).ClearOrder(workingDir, order)
		if err != nil || write.Changed || !strings.HasSuffix(write.Detail, "to clear") {
			t.Errorf("ClearOrder of an absent order = %+v, %v, want unchanged", write, err)
		}
	}
}

// Each refusal leaves the file exactly as it was and says why in words a person can act on.
func TestSetOrderRefuses(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings string
		order    Order
		names    []string
		want     string
	}{
		{
			name: "review naming an agent the list does not define", settings: orderedSettings,
			order: ReviewOrder(), names: []string{"architect", "reviewer"},
			want: "the order must name agents from the list, each once (agents: subagent, architect): not in the list: reviewer",
		},
		{
			name: "consult naming an agent the list does not define", settings: orderedSettings,
			order: ConsultOrder(), names: []string{"reviewer"}, want: "not in the list: reviewer",
		},
		{
			name: "a harness naming an agent the list does not define", settings: orderedSettings,
			order: HarnessOrder("claude"), names: []string{"reviewer"}, want: "not in the list: reviewer",
		},
		{
			name: "review naming an agent twice", settings: orderedSettings,
			order: ReviewOrder(), names: []string{"architect", "subagent", "architect"}, want: "named twice: architect",
		},
		{
			name: "consult naming an agent twice", settings: orderedSettings,
			order: ConsultOrder(), names: []string{"subagent", "subagent"}, want: "named twice: subagent",
		},
		{
			name: "a harness naming an agent twice", settings: orderedSettings,
			order: HarnessOrder("codex"), names: []string{"subagent", "subagent"}, want: "named twice: subagent",
		},
		{
			name: "an order with no agents", settings: orderedSettings,
			order: ConsultOrder(), names: nil, want: errEmptyOrder.Error(),
		},
		{
			name: "a harness codefall cannot set up", settings: orderedSettings,
			order: HarnessOrder("cursor"), names: []string{"subagent"},
			want: `harness "cursor" is not supported yet (supported: agy, claude, codex, muse, opencode)`,
		},
		{
			name: "settings that are already invalid", settings: `{"version": 1, "harnesses": ["claude"]}`,
			order: ConsultOrder(), names: []string{"subagent"},
			want: ".codefall/settings.json is not valid, so nothing was changed: tracker: missing",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := project(tc.settings)

			_, err := NewConfig(files).SetOrder(workingDir, tc.order, tc.names)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("SetOrder error = %v, want it to say %q", err, tc.want)
			}

			if len(files.writes) != 0 || string(files.files[settingsFull]) != tc.settings {
				t.Errorf("wrote %q, want the file left as it was", files.writes)
			}
		})
	}
}

func TestClearOrderRefuses(t *testing.T) {
	files := project(orderedSettings)

	_, err := NewConfig(files).ClearOrder(workingDir, HarnessOrder("cursor"))
	if err == nil || !strings.Contains(err.Error(), `harness "cursor" is not supported yet`) {
		t.Errorf("ClearOrder of an unknown harness = %v, want the harness module's refusal", err)
	}

	_, err = NewConfig(newFakeFileSystem(nil)).ClearOrder(workingDir, ConsultOrder())
	if !errors.Is(err, errNotSetUp) {
		t.Errorf("ClearOrder with no settings = %v, want %v", err, errNotSetUp)
	}

	if len(files.writes) != 0 {
		t.Errorf("wrote %q, want nothing", files.writes)
	}
}

// Once each order has been changed by its command, the agent it named can go.
func TestRemoveAgentAfterItsOrdersAreCleared(t *testing.T) {
	files := project(referencedSettings)
	config := NewConfig(files)

	if _, err := config.SetOrder(workingDir, ReviewOrder(), []string{"subagent"}); err != nil {
		t.Fatalf("SetOrder: %v", err)
	}

	if _, err := config.ClearOrder(workingDir, HarnessOrder("claude")); err != nil {
		t.Fatalf("ClearOrder: %v", err)
	}

	if _, err := config.RemoveAgent(workingDir, "architect"); err != nil {
		t.Fatalf("RemoveAgent: %v", err)
	}
}
