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

// The agents a project reaches for when a verb needs another reader: a reviewer, or a second
// opinion on a decision the run cannot settle (ADR-009). The top-level list defines them and their
// default order; a use may carry an `agents` key of its own holding an ordered subset of the names,
// and the per-harness override does the same for the harness a session is running in.
const (
	// FieldAgents is the top-level list: each entry a name, a harness, and optionally a model.
	FieldAgents = "agents"
	// FieldAgentsByHarness is the per-harness override, keyed by the harness running the session and
	// holding an ordered subset of the names above. A session that cannot say which harness it is in
	// falls through to the top-level order.
	FieldAgentsByHarness = "agentsByHarness"
	// FieldReviewAgents is the review block's own order, an ordered subset of the names above. The
	// same word means the same thing wherever it appears.
	FieldReviewAgents = "agents"
	// The three fields of one agent.
	FieldAgentName    = "name"
	FieldAgentHarness = "harness"
	FieldAgentModel   = "model"
	// HarnessCurrent names the harness running the session, whatever it is, so one checked-in entry
	// means Claude Code's subagent in one person's session and Muse's in another's. It is always
	// runnable, which is what makes it the floor of an order.
	HarnessCurrent = "current"
	// DefaultAgentName is the one agent a project has when it has chosen none.
	DefaultAgentName = "subagent"
	// AgentNamePattern is how an agent may be named: a lower-case slug, because the name is typed
	// after `via=` and read back in a report.
	AgentNamePattern = `^[a-z0-9][a-z0-9-]*$`
)

var agentNameRegexp = regexp.MustCompile(AgentNamePattern)

// Agent is one entry of the list: the name a verb or a person refers to it by, the harness that
// runs it, and the model to ask that harness for, when the project has chosen one.
type Agent struct {
	Name    string
	Harness string
	Model   mo.Option[string]
}

// DefaultAgents is the list a project has when settings carry none: this harness's own subagent,
// under the name the review skill has always used for it. init writes it explicitly into a new
// project's settings, and every reader takes it when the field is absent, so the two agree.
func DefaultAgents() []Agent {
	return []Agent{{Name: DefaultAgentName, Harness: HarnessCurrent, Model: mo.None[string]()}}
}

// AgentHarnesses returns every value an agent's harness field accepts, sorted: the harnesses
// codefall can set up, and current.
func AgentHarnesses() []string {
	return slices.Sorted(slices.Values(append(harness.All(), HarnessCurrent)))
}

// Agents returns the agents a document defines, in the order it defines them, and the default when
// the field is absent or empty. A list Validate would reject is read as absent too: the caller has
// already been told what is wrong with it, and a list nothing can act on is no list.
func Agents(doc Document) []Agent {
	value, present := lookup(doc, FieldAgents)
	if !present || isAgents(value) != "" {
		return DefaultAgents()
	}

	entries, _ := value.([]any)
	if len(entries) == 0 {
		return DefaultAgents()
	}

	agents := make([]Agent, 0, len(entries))

	for _, entry := range entries {
		fields, _ := entry.(map[string]any)
		name, _ := fields[FieldAgentName].(string)
		runs, _ := fields[FieldAgentHarness].(string)

		agent := Agent{Name: name, Harness: runs, Model: mo.None[string]()}

		if model, ok := lookup(fields, FieldAgentModel); ok {
			text, _ := model.(string)
			agent.Model = mo.Some(text)
		}

		agents = append(agents, agent)
	}

	return agents
}

// AgentNames returns the names of a list of agents, in order.
func AgentNames(agents []Agent) []string {
	names := make([]string, 0, len(agents))
	for _, agent := range agents {
		names = append(names, agent.Name)
	}

	return names
}

// ReviewAgents returns the review block's own order when the document carries one that Validate
// accepts, and None when it does not, which means the top-level order applies.
func ReviewAgents(doc Document) mo.Option[[]string] {
	block, ok := lookupObject(doc, BlockReview)
	if !ok {
		return mo.None[[]string]()
	}

	value, present := lookup(block, FieldReviewAgents)
	if !present || isNameList(value) != "" {
		return mo.None[[]string]()
	}

	return mo.Some(nameList(value))
}

// AgentsByHarness returns the per-harness override, keyed by harness, with only the entries
// Validate would accept. A document without the field yields an empty map.
func AgentsByHarness(doc Document) map[string][]string {
	orders := map[string][]string{}

	block, ok := lookupObject(doc, FieldAgentsByHarness)
	if !ok {
		return orders
	}

	for name, value := range block {
		if harness.SkillsDir(name).IsAbsent() || isNameList(value) != "" {
			continue
		}

		orders[name] = nameList(value)
	}

	return orders
}

// HasCurrent reports whether an order names an agent that runs on the current harness, which is
// what makes the order end somewhere a run can always start.
func HasCurrent(agents []Agent, order []string) bool {
	for _, name := range order {
		for _, agent := range agents {
			if agent.Name == name && agent.Harness == HarnessCurrent {
				return true
			}
		}
	}

	return false
}

// isAgents accepts the top-level list: any number of agents, each an object naming an agent by a
// slug nobody else in the list uses and a harness codefall can start or current, with a model when
// the project has chosen one. An empty list is accepted and means the default, the same as an
// absent one.
func isAgents(v any) string {
	entries, ok := v.([]any)
	if !ok {
		return "must be an array of agents"
	}

	seen := map[string]bool{}

	for i, entry := range entries {
		fields, ok := entry.(map[string]any)
		if !ok {
			return fmt.Sprintf("[%d]: must be an object", i)
		}

		name, present := lookup(fields, FieldAgentName)
		if !present {
			return fmt.Sprintf("[%d].%s: missing", i, FieldAgentName)
		}

		text, isText := name.(string)
		if !isText || !agentNameRegexp.MatchString(text) {
			return fmt.Sprintf("[%d].%s: must match %s", i, FieldAgentName, AgentNamePattern)
		}

		if seen[text] {
			return fmt.Sprintf("names %q twice", text)
		}

		seen[text] = true

		runs, present := lookup(fields, FieldAgentHarness)
		if !present {
			return fmt.Sprintf("[%d].%s: missing", i, FieldAgentHarness)
		}

		if reason := isAgentHarness(runs); reason != "" {
			return fmt.Sprintf("[%d].%s: %s", i, FieldAgentHarness, reason)
		}

		if model, present := lookup(fields, FieldAgentModel); present {
			if _, isText := model.(string); !isText {
				return fmt.Sprintf("[%d].%s: must be a string", i, FieldAgentModel)
			}
		}
	}

	return ""
}

// isAgentHarness accepts what runs an agent: a harness under the name it has now, or current. A
// former spelling is refused here, unlike in the harnesses field: the agents list is newer than
// the rename, so nothing checked in can carry one.
func isAgentHarness(v any) string {
	name, ok := v.(string)
	if !ok {
		return "must be a string"
	}

	if name == HarnessCurrent || harness.SkillsDir(name).IsPresent() {
		return ""
	}

	return unknownValue(name, AgentHarnesses())
}

// isNameList accepts an order: agent names, each once. Whether each names an agent the list
// defines is checked across the document by agentReferences, because a field's own check sees
// only its value.
func isNameList(v any) string {
	values, ok := v.([]any)
	if !ok {
		return "must be an array of agent names"
	}

	seen := map[string]bool{}

	for _, value := range values {
		name, ok := value.(string)
		if !ok {
			return "must be an array of agent names"
		}

		if seen[name] {
			return fmt.Sprintf("names %q twice", name)
		}

		seen[name] = true
	}

	return ""
}

func nameList(v any) []string {
	values, _ := v.([]any)
	names := make([]string, 0, len(values))

	for _, value := range values {
		name, _ := value.(string)
		names = append(names, name)
	}

	return names
}

// agentReferences checks what no single field can: that every order names agents the top-level
// list defines, and that the per-harness override is keyed by harnesses codefall knows. It runs
// only when the top-level list was acceptable, because an order checked against a list that was
// itself refused would say the same thing twice.
func agentReferences(doc Document, rejected map[string]bool) []string {
	if rejected[FieldAgents] {
		return nil
	}

	defined := AgentNames(Agents(doc))

	var problems []string

	if block, ok := lookupObject(doc, BlockReview); ok && !rejected[BlockReview] {
		if value, present := lookup(block, FieldReviewAgents); present && isNameList(value) == "" {
			problems = append(problems, undefinedNames(BlockReview+"."+FieldReviewAgents, nameList(value), defined)...)
		}
	}

	block, ok := lookupObject(doc, FieldAgentsByHarness)
	if !ok || rejected[FieldAgentsByHarness] {
		return problems
	}

	for _, key := range slices.Sorted(maps.Keys(block)) {
		path := FieldAgentsByHarness + "." + key

		if harness.SkillsDir(key).IsAbsent() {
			problems = append(problems, path+": "+unknownValue(key, harness.All()))

			continue
		}

		if reason := isNameList(block[key]); reason != "" {
			problems = append(problems, path+": "+reason)

			continue
		}

		problems = append(problems, undefinedNames(path, nameList(block[key]), defined)...)
	}

	return problems
}

// undefinedNames is one problem per name an order uses that the list does not define.
func undefinedNames(path string, order, defined []string) []string {
	var problems []string

	for _, name := range order {
		if !slices.Contains(defined, name) {
			problems = append(problems, fmt.Sprintf("%s: names %q, which %s does not define", path, name, FieldAgents))
		}
	}

	return problems
}

// lookupObject is lookup for a key that has to hold an object.
func lookupObject(doc map[string]any, name string) (map[string]any, bool) {
	value, present := lookup(doc, name)
	if !present {
		return nil, false
	}

	block, ok := value.(map[string]any)

	return block, ok
}

// DescribeAgent is "architect (codex, gpt-5-codex)" or "subagent (current)": how a report names an
// agent so a reader can match it to the settings.
func DescribeAgent(agent Agent) string {
	if model, ok := agent.Model.Get(); ok {
		return fmt.Sprintf("%s (%s, %s)", agent.Name, agent.Harness, model)
	}

	return fmt.Sprintf("%s (%s)", agent.Name, agent.Harness)
}

// DescribeAgents is the list, comma separated, in order.
func DescribeAgents(agents []Agent) string {
	parts := make([]string, 0, len(agents))
	for _, agent := range agents {
		parts = append(parts, DescribeAgent(agent))
	}

	return strings.Join(parts, ", ")
}
