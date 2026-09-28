package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// Order names one of the narrower orders a write changes: review's own, consult's own, or the order a
// session running in one harness walks (ADR-009). Presentation builds one with ReviewOrder,
// ConsultOrder, or HarnessOrder; a harness name is checked when the order is written.
type Order struct {
	block   string
	harness string
}

// ReviewOrder is review.agents.
func ReviewOrder() Order {
	return Order{block: settings.BlockReview}
}

// ConsultOrder is consult.agents.
func ConsultOrder() Order {
	return Order{block: settings.BlockConsult}
}

// HarnessOrder is agentsByHarness.<name>.
func HarnessOrder(name string) Order {
	return Order{block: settings.FieldAgentsByHarness, harness: name}
}

// errEmptyOrder refuses an order with no agents in it, which would leave a use nothing to walk. The
// way back to the wider order is to clear this one.
var errEmptyOrder = errors.New("an order needs at least one agent; clear it instead to walk the wider order")

// orderPlace is where one narrower order sits in the settings, and the command that changes it.
type orderPlace struct {
	// block is the top-level key, and key the one inside it that holds the order.
	block string
	key   string
	// needsBlock says the block carries required fields of its own, so the order is added only to a
	// block that is already there rather than to one this write would have to invent.
	needsBlock bool
	// dropEmpty says the block means nothing without an entry, so clearing the last one removes it.
	dropEmpty bool
}

func (p orderPlace) path() string {
	return p.block + "." + p.key
}

// command is the `codefall config` command that sets or clears this order.
func (p orderPlace) command() string {
	if p.block == settings.FieldAgentsByHarness {
		return "codefall config agents for " + p.key
	}

	return "codefall config " + p.block + " agents"
}

// read is the order as the settings hold it, None when they hold none. It is called only on settings
// the module has accepted, so a present order is a list of names.
func (p orderPlace) read(doc settings.Document) mo.Option[[]string] {
	block, _ := doc[p.block].(map[string]any)

	values, ok := block[p.key].([]any)
	if !ok {
		return mo.None[[]string]()
	}

	names := make([]string, 0, len(values))

	for _, value := range values {
		name, _ := value.(string)
		names = append(names, name)
	}

	return mo.Some(names)
}

// place resolves an order to where it sits, refusing a harness codefall cannot set up in the harness
// module's own words. A former spelling of a harness resolves to the name it has now.
func (o Order) place() (orderPlace, error) {
	switch o.block {
	case settings.BlockReview:
		return orderPlace{block: settings.BlockReview, key: settings.FieldReviewAgents, needsBlock: true}, nil
	case settings.BlockConsult:
		return orderPlace{block: settings.BlockConsult, key: settings.FieldConsultAgents}, nil
	}

	name, err := harness.Parse(o.harness)
	if err != nil {
		return orderPlace{}, err
	}

	return orderPlace{block: settings.FieldAgentsByHarness, key: name, dropEmpty: true}, nil
}

// allPlaces is every narrower order the settings can carry, in the order Referrers reports them.
func allPlaces() []orderPlace {
	places := []orderPlace{
		{block: settings.BlockReview, key: settings.FieldReviewAgents, needsBlock: true},
		{block: settings.BlockConsult, key: settings.FieldConsultAgents},
	}

	for _, name := range harness.All() {
		places = append(places, orderPlace{block: settings.FieldAgentsByHarness, key: name, dropEmpty: true})
	}

	return places
}

// SetOrder sets one narrower order to the names given, in that order. Every name must be an agent the
// list defines, each once. The consult block and agentsByHarness are created when absent; the review
// block is not, because it carries a choice of its own the project has to make first. Only the
// order's own text changes, and an order already equal to the one given is left alone.
func (c *Config) SetOrder(dir string, order Order, names []string) (domain.Write, error) {
	place, err := order.place()
	if err != nil {
		return domain.Write{}, err
	}

	project, err := c.startOrder(dir)
	if err != nil {
		return domain.Write{}, err
	}

	if len(names) == 0 {
		return domain.Write{}, errEmptyOrder
	}

	if err := domain.CheckSubset(settings.AgentNames(settings.Agents(project.doc)), names); err != nil {
		return domain.Write{}, err
	}

	joined := strings.Join(names, ", ")

	if current, ok := place.read(project.doc).Get(); ok && slices.Equal(current, names) {
		return domain.Unchanged(fmt.Sprintf("%s already sets %s to %s", settingsName, place.path(), joined)), nil
	}

	body, err := withOrder(project, place, names)
	if err != nil {
		return domain.Write{}, err
	}

	if err := c.writeSettings(dir, body); err != nil {
		return domain.Write{}, err
	}

	return domain.Changed(fmt.Sprintf("set %s in %s: %s", place.path(), settingsName, joined)), nil
}

// ClearOrder removes one narrower order, so the use or the harness walks the wider order again. The
// consult block stays when it is left empty, which the schema allows; agentsByHarness goes with its
// last entry.
func (c *Config) ClearOrder(dir string, order Order) (domain.Write, error) {
	place, err := order.place()
	if err != nil {
		return domain.Write{}, err
	}

	project, err := c.startOrder(dir)
	if err != nil {
		return domain.Write{}, err
	}

	if place.read(project.doc).IsAbsent() {
		return domain.Unchanged(fmt.Sprintf("%s has no %s to clear", settingsName, place.path())), nil
	}

	body, empty, err := withoutNestedField(project.data, place.block, place.key)
	if err != nil {
		return domain.Write{}, fmt.Errorf("decode %s: %w", settingsName, err)
	}

	detail := fmt.Sprintf("removed %s from %s", place.path(), settingsName)

	if empty && place.dropEmpty {
		if body, err = withoutField(body, place.block); err != nil {
			return domain.Write{}, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		detail += ", and the empty " + place.block + " with it"
	}

	if err := c.writeSettings(dir, body); err != nil {
		return domain.Write{}, err
	}

	return domain.Changed(detail), nil
}

// startOrder is what every order write starts from: settings that exist and that the module accepts.
func (c *Config) startOrder(dir string) (projectSettings, error) {
	project, err := c.readSettings(dir)
	if err != nil {
		return projectSettings{}, err
	}

	if err := validSettings(project.doc, settingsName+" is not valid, so nothing was changed"); err != nil {
		return projectSettings{}, err
	}

	return project, nil
}

// withOrder is the settings' text with one order set: the order's value replaced in a block that is
// there, the order added to it when it holds none, and a new block holding only the order when there
// is none and the order may have one made for it.
func withOrder(project projectSettings, place orderPlace, names []string) ([]byte, error) {
	if _, isObject := project.doc[place.block].(map[string]any); isObject {
		inner, err := nestedObject(project.data, place.block)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		existing, held := inner.last(place.key)
		multiline := held && bytes.ContainsRune(existing.raw, '\n')

		body, err := withNestedField(project.data, place.block, place.key, func(indent string) ([]byte, error) {
			return renderNames(names, multiline, indent)
		})
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", settingsName, err)
		}

		return body, nil
	}

	if place.needsBlock {
		return nil, fmt.Errorf("%s has no %s block, and one must carry %s, which is the project's choice; "+
			"add the block first, then set its order", settingsName, place.block,
			strings.Join(settings.RequiredReviewFields(), ", "))
	}

	body, err := withField(project.data, place.block, func(indent string) ([]byte, error) {
		inner := indent + "  "

		key, err := json.Marshal(place.key)
		if err != nil {
			return nil, err
		}

		value, err := renderNames(names, false, inner)
		if err != nil {
			return nil, err
		}

		return slices.Concat([]byte("{\n"+inner), key, []byte(": "), value, []byte("\n"+indent+"}")), nil
	})
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", settingsName, err)
	}

	return body, nil
}

// renderNames lays an order out on one line, or one name per line one level in from the key when the
// order it replaces spanned lines.
func renderNames(names []string, multiline bool, indent string) ([]byte, error) {
	items := make([][]byte, 0, len(names))

	for _, name := range names {
		item, err := json.Marshal(name)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if !multiline {
		return slices.Concat([]byte("["), bytes.Join(items, []byte(", ")), []byte("]")), nil
	}

	inner := indent + "  "

	return slices.Concat([]byte("[\n"+inner), bytes.Join(items, []byte(",\n"+inner)), []byte("\n"+indent+"]")), nil
}

// changeOrders is, for each order that still names an agent, the command that changes it: set to the
// rest of its names when any are left, and cleared when none are.
func changeOrders(doc settings.Document, name string, referrers []string) []string {
	places := allPlaces()
	commands := make([]string, 0, len(referrers))

	for _, path := range referrers {
		at := slices.IndexFunc(places, func(p orderPlace) bool { return p.path() == path })
		if at < 0 {
			continue
		}

		place := places[at]
		rest := slices.DeleteFunc(place.read(doc).OrElse(nil), func(other string) bool { return other == name })

		if len(rest) == 0 {
			commands = append(commands, place.command()+" --clear")

			continue
		}

		commands = append(commands, place.command()+" "+strings.Join(rest, " ")+" (or --clear)")
	}

	return commands
}
