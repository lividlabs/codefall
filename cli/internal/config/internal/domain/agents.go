package domain

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// InsertAt is where a new agent goes in an order of names: straight before one name, straight after
// one, or at the end when neither is given. A name to place it by that the order does not hold is an
// error naming the ones it does. Asking for both is presentation's to refuse; before wins here.
func InsertAt(order []string, before, after mo.Option[string]) (int, error) {
	if name, ok := before.Get(); ok {
		at := slices.Index(order, name)
		if at < 0 {
			return 0, NotInList(name, order)
		}

		return at, nil
	}

	if name, ok := after.Get(); ok {
		at := slices.Index(order, name)
		if at < 0 {
			return 0, NotInList(name, order)
		}

		return at + 1, nil
	}

	return len(order), nil
}

// CheckOrder refuses a proposed order unless it names every agent in the current one exactly once.
// Every problem is named at once — the agents left out, the names the list does not define, and the
// names given twice — because someone fixing the command line wants the whole list.
func CheckOrder(current, proposed []string) error {
	var problems []string

	var missing []string

	for _, name := range current {
		if !slices.Contains(proposed, name) {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		problems = append(problems, "missing "+strings.Join(missing, ", "))
	}

	unknown, repeated := misnamed(current, proposed)

	if len(unknown) > 0 {
		problems = append(problems, "not in the list: "+strings.Join(unknown, ", "))
	}

	if len(repeated) > 0 {
		problems = append(problems, "named twice: "+strings.Join(repeated, ", "))
	}

	if len(problems) == 0 {
		return nil
	}

	return fmt.Errorf("the order must name every agent exactly once (agents: %s): %s",
		strings.Join(current, ", "), strings.Join(problems, "; "))
}

// CheckSubset refuses a proposed narrower order unless every name in it is an agent the list defines
// and none is given twice. It need not name every agent: a use's own order, or a harness's, is the
// subset it walks. Every problem is named at once, as CheckOrder does.
func CheckSubset(defined, proposed []string) error {
	unknown, repeated := misnamed(defined, proposed)

	var problems []string

	if len(unknown) > 0 {
		problems = append(problems, "not in the list: "+strings.Join(unknown, ", "))
	}

	if len(repeated) > 0 {
		problems = append(problems, "named twice: "+strings.Join(repeated, ", "))
	}

	if len(problems) == 0 {
		return nil
	}

	return fmt.Errorf("the order must name agents from the list, each once (agents: %s): %s",
		strings.Join(defined, ", "), strings.Join(problems, "; "))
}

// misnamed is what a proposed order gets wrong beside what it leaves out: the names the list does
// not define, and the names given twice, each reported once in the order first seen.
func misnamed(defined, proposed []string) (unknown, repeated []string) {
	seen := map[string]bool{}

	for _, name := range proposed {
		switch {
		case !slices.Contains(defined, name):
			if !slices.Contains(unknown, name) {
				unknown = append(unknown, name)
			}
		case seen[name]:
			if !slices.Contains(repeated, name) {
				repeated = append(repeated, name)
			}
		}

		seen[name] = true
	}

	return unknown, repeated
}

// Referrers is every order in a settings document that names an agent, as the path a person would
// edit: review.agents, consult.agents, then agentsByHarness.<harness> in harness order. Removing an
// agent one of them still names would leave that order pointing at nothing.
func Referrers(doc settings.Document, name string) []string {
	var paths []string

	for _, use := range []struct {
		path  string
		order mo.Option[[]string]
	}{
		{settings.BlockReview + "." + settings.FieldReviewAgents, settings.ReviewAgents(doc)},
		{settings.BlockConsult + "." + settings.FieldConsultAgents, settings.ConsultAgents(doc)},
	} {
		if order, ok := use.order.Get(); ok && slices.Contains(order, name) {
			paths = append(paths, use.path)
		}
	}

	byHarness := settings.AgentsByHarness(doc)

	for _, key := range slices.Sorted(maps.Keys(byHarness)) {
		if slices.Contains(byHarness[key], name) {
			paths = append(paths, settings.FieldAgentsByHarness+"."+key)
		}
	}

	return paths
}

// StillNamed is the refusal to remove an agent an order still names: one sentence naming the orders,
// one saying where to change them. Nothing in it is computed beyond the names, so it reads the same
// whether one order or five still point at the agent. The second sentence ends without a full stop
// because the command's error renderer adds one.
func StillNamed(name string, referrers []string) error {
	orders := "that order"
	if len(referrers) > 1 {
		orders = "those orders"
	}

	return fmt.Errorf("%s is named in %s. Change %s first, in codefall config, then remove it",
		name, joinAnd(referrers), orders)
}

// joinAnd joins names the way a sentence does: "a", "a and b", "a, b, and c".
func joinAnd(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
	}
}

// NotInList is the error for a name the list does not hold, with the names it does.
func NotInList(name string, order []string) error {
	return fmt.Errorf("no agent named %q in the list (agents: %s)", name, strings.Join(order, ", "))
}
