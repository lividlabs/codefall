package settings

import (
	"fmt"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
)

// The agents a project reaches for when a verb needs another reader: a reviewer, or a second
// opinion on a decision the run cannot settle (ADR-009.2). The list is keyed by the harness the
// session is running in, the active agent, because which agent to reach for is a question asked
// from inside a harness: a session in Muse may want Claude to review, a session in Claude Code may
// want Codex. Each entry holds one ordered list per feature, review and consult.
const (
	// FieldAgents is the top-level array of entries, one per active agent.
	FieldAgents = "agents"
	// FieldActiveAgent is the harness an entry is for: one of the five harness names, or default,
	// which covers every harness without an entry of its own.
	FieldActiveAgent = "activeAgent"
	// FeatureReview and FeatureConsult are the two lists an entry holds, under the feature's name.
	FeatureReview  = "review"
	FeatureConsult = "consult"
	// The two fields of one agent in a list.
	FieldAgentHarness = "harness"
	FieldAgentModel   = "model"
	// ActiveDefault is the entry every active agent falls back to.
	ActiveDefault = "default"
	// HarnessCurrent, in a list, names the harness running the session, whatever it is, so one
	// checked-in entry means Claude Code's subagent in one person's session and Muse's in another's.
	// It is always runnable, which is what makes it the floor of a list.
	HarnessCurrent = "current"
)

// Agent is one agent in a list: the harness that runs it, and the model to ask that harness for,
// when the project has chosen one.
type Agent struct {
	Harness string
	Model   mo.Option[string]
}

// Entry is what one active agent reaches for: its review list and its consult list, each absent
// when the entry leaves that feature to the default entry.
type Entry struct {
	ActiveAgent string
	Review      mo.Option[[]Agent]
	Consult     mo.Option[[]Agent]
}

// List returns the entry's list for a feature, absent when the entry has none.
func (e Entry) List(feature string) mo.Option[[]Agent] {
	switch feature {
	case FeatureReview:
		return e.Review
	case FeatureConsult:
		return e.Consult
	}

	return mo.None[[]Agent]()
}

// Features returns the two features an entry holds a list for, in the order the file writes them.
func Features() []string {
	return []string{FeatureReview, FeatureConsult}
}

// CurrentAgent is the one agent every list falls back to: a subagent of the harness running the
// session.
func CurrentAgent() Agent {
	return Agent{Harness: HarnessCurrent, Model: mo.None[string]()}
}

// DefaultAgents is the list a project has when settings carry none: one default entry whose two
// lists each hold the current harness's own subagent, which is what every verb did before the list
// existed. init writes it explicitly into a new project's settings, and every reader takes it when
// the field is absent, so the two agree.
func DefaultAgents() []Entry {
	return []Entry{{
		ActiveAgent: ActiveDefault,
		Review:      mo.Some([]Agent{CurrentAgent()}),
		Consult:     mo.Some([]Agent{CurrentAgent()}),
	}}
}

// ActiveAgents returns every value the activeAgent field accepts, sorted: the harnesses codefall
// can set up, and default.
func ActiveAgents() []string {
	return slices.Sorted(slices.Values(append(harness.All(), ActiveDefault)))
}

// AgentHarnesses returns every value an agent's harness field accepts, sorted: the harnesses
// codefall can set up, and current.
func AgentHarnesses() []string {
	return slices.Sorted(slices.Values(append(harness.All(), HarnessCurrent)))
}

// ParseActiveAgent returns the active agent name when it is one the format knows, and an error
// naming the ones it does when it is not. A former spelling of a harness resolves to its current
// name, as the harness module has it.
func ParseActiveAgent(name string) (string, error) {
	if name == ActiveDefault {
		return name, nil
	}

	if parsed, err := harness.Parse(name); err == nil {
		return parsed, nil
	}

	return "", fmt.Errorf("active agent %q is not one codefall knows (known: %s)", name, strings.Join(ActiveAgents(), ", "))
}

// ParseAgent reads an agent from the form a person types, `harness` or `harness:model`: the form
// `via=` takes, and the form a report writes back. A former spelling of a harness resolves to its
// current name. A name that is not a harness is accepted when it is written the way a harnessConfig
// key is, since whether the document holds that key is the document's check to make; a name written
// any other way can be nothing, and is refused here.
func ParseAgent(text string) (Agent, error) {
	name, model, hasModel := strings.Cut(text, ":")

	if hasModel && model == "" {
		return Agent{}, fmt.Errorf("agent %q names an empty model", text)
	}

	if name == HarnessCurrent {
		return Agent{Harness: name, Model: modelOf(model, hasModel)}, nil
	}

	if parsed, err := harness.Parse(name); err == nil {
		return Agent{Harness: parsed, Model: modelOf(model, hasModel)}, nil
	}

	if !harnessKeyRegexp.MatchString(name) {
		return Agent{}, fmt.Errorf("agent %q: harness %q is not a harness (%s), %s, or a %s key, "+
			"which is lowercase letters and digits joined by hyphens",
			text, name, strings.Join(harness.All(), ", "), HarnessCurrent, FieldHarnessConfig)
	}

	return Agent{Harness: name, Model: modelOf(model, hasModel)}, nil
}

func modelOf(model string, has bool) mo.Option[string] {
	if !has {
		return mo.None[string]()
	}

	return mo.Some(model)
}

// Agents returns the entries a document defines, in the order it defines them, and the default when
// the field is absent or empty. A list Validate would reject is read as absent too: the caller has
// already been told what is wrong with it, and a list nothing can act on is no list.
func Agents(doc Document) []Entry {
	value, present := lookup(doc, FieldAgents)
	if !present || isAgents(value) != "" || agentHarnessProblem(doc) != "" {
		return DefaultAgents()
	}

	items, _ := value.([]any)
	if len(items) == 0 {
		return DefaultAgents()
	}

	entries := make([]Entry, 0, len(items))

	for _, item := range items {
		fields, _ := item.(map[string]any)
		active, _ := fields[FieldActiveAgent].(string)

		entry := Entry{ActiveAgent: active, Review: mo.None[[]Agent](), Consult: mo.None[[]Agent]()}

		if value, ok := lookup(fields, FeatureReview); ok {
			entry.Review = mo.Some(agentList(value))
		}

		if value, ok := lookup(fields, FeatureConsult); ok {
			entry.Consult = mo.Some(agentList(value))
		}

		entries = append(entries, entry)
	}

	return entries
}

// EntryFor returns the entry for an active agent, and None when the document has none for it.
func EntryFor(doc Document, active string) mo.Option[Entry] {
	for _, entry := range Agents(doc) {
		if entry.ActiveAgent == active {
			return mo.Some(entry)
		}
	}

	return mo.None[Entry]()
}

// Order resolves the list a session running in the active agent walks for a feature: the active
// agent's own list, else the default entry's list, else the current harness's subagent alone. It
// never comes back empty.
func Order(doc Document, active, feature string) []Agent {
	if entry, ok := EntryFor(doc, active).Get(); ok {
		if agents, has := entry.List(feature).Get(); has {
			return agents
		}
	}

	if entry, ok := EntryFor(doc, ActiveDefault).Get(); ok {
		if agents, has := entry.List(feature).Get(); has {
			return agents
		}
	}

	return []Agent{CurrentAgent()}
}

// HasCurrent reports whether a list names the current harness, which is what makes the list end
// somewhere a run can always start.
func HasCurrent(agents []Agent) bool {
	for _, agent := range agents {
		if agent.Harness == HarnessCurrent {
			return true
		}
	}

	return false
}

// ActiveAgentNames returns the active agents a list of entries is for, in order.
func ActiveAgentNames(entries []Entry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.ActiveAgent)
	}

	return names
}

// agentList reads a list Validate has accepted.
func agentList(value any) []Agent {
	items, _ := value.([]any)
	agents := make([]Agent, 0, len(items))

	for _, item := range items {
		fields, _ := item.(map[string]any)
		runs, _ := fields[FieldAgentHarness].(string)

		agent := Agent{Harness: runs, Model: mo.None[string]()}

		if model, ok := lookup(fields, FieldAgentModel); ok {
			text, _ := model.(string)
			agent.Model = mo.Some(text)
		}

		agents = append(agents, agent)
	}

	return agents
}

// isAgents accepts the top-level array: any number of entries, each an object naming an active
// agent nobody else in the array names, with a review list, a consult list, or both, each a
// non-empty array of agents whose harness is written. An empty array is accepted and means the
// default, the same as an absent one. What a harness may name is checked against the whole document
// afterwards, by agentHarnessProblem, because a harnessConfig key is one of the answers.
func isAgents(v any) string {
	items, ok := v.([]any)
	if !ok {
		return "must be an array of entries, one per active agent"
	}

	seen := map[string]bool{}

	for i, item := range items {
		fields, ok := item.(map[string]any)
		if !ok {
			return fmt.Sprintf("[%d]: must be an object", i)
		}

		active, present := lookup(fields, FieldActiveAgent)
		if !present {
			return fmt.Sprintf("[%d].%s: missing", i, FieldActiveAgent)
		}

		name, isText := active.(string)
		if !isText {
			return fmt.Sprintf("[%d].%s: must be a string", i, FieldActiveAgent)
		}

		if name != ActiveDefault && harness.SkillsDir(name).IsAbsent() {
			return fmt.Sprintf("[%d].%s: %s", i, FieldActiveAgent, unknownValue(name, ActiveAgents()))
		}

		if seen[name] {
			return fmt.Sprintf("names %s %q twice", FieldActiveAgent, name)
		}

		seen[name] = true

		for _, feature := range Features() {
			value, present := lookup(fields, feature)
			if !present {
				continue
			}

			if reason := isAgentList(value); reason != "" {
				return fmt.Sprintf("[%d].%s%s", i, feature, reason)
			}
		}
	}

	return ""
}

// isAgentList accepts one feature's list: at least one agent, each an object with a harness written
// as a string, and a model when the project has chosen one. An empty list is refused rather than
// read as absent: leaving the key out is how an entry defers to the default, and an empty list
// would say the same thing a second way.
func isAgentList(v any) string {
	items, ok := v.([]any)
	if !ok {
		return ": must be an array of agents"
	}

	if len(items) == 0 {
		return ": must name at least one agent; leave the key out to use the default entry's list"
	}

	for i, item := range items {
		fields, ok := item.(map[string]any)
		if !ok {
			return fmt.Sprintf("[%d]: must be an object", i)
		}

		runs, present := lookup(fields, FieldAgentHarness)
		if !present {
			return fmt.Sprintf("[%d].%s: missing", i, FieldAgentHarness)
		}

		if _, isText := runs.(string); !isText {
			return fmt.Sprintf("[%d].%s: must be a string", i, FieldAgentHarness)
		}

		if model, present := lookup(fields, FieldAgentModel); present {
			if _, isText := model.(string); !isText {
				return fmt.Sprintf("[%d].%s: must be a string", i, FieldAgentModel)
			}
		}
	}

	return ""
}

// DescribeAgent is the form a person types and a report reads back: "codex:gpt-5-codex", or
// "current" for the harness running the session.
func DescribeAgent(agent Agent) string {
	if model, ok := agent.Model.Get(); ok {
		return agent.Harness + ":" + model
	}

	return agent.Harness
}

// DescribeAgents is a list, comma separated, in order.
func DescribeAgents(agents []Agent) string {
	parts := make([]string, 0, len(agents))
	for _, agent := range agents {
		parts = append(parts, DescribeAgent(agent))
	}

	return strings.Join(parts, ", ")
}
