package application

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// NewAgent is what `codefall config agents add` hands the use case: the agent, and where in the list
// it goes. Before and After are each None unless the flag was given; with neither, it goes last.
type NewAgent struct {
	Name    string
	Harness string
	Model   mo.Option[string]
	Before  mo.Option[string]
	After   mo.Option[string]
}

// agentDocument is one entry of the list in the file's shape, for an entry this command writes. The
// field order is the one init writes.
type agentDocument struct {
	Name    string            `json:"name"`
	Harness string            `json:"harness"`
	Model   mo.Option[string] `json:"model,omitzero"`
}

// agentEntry is one entry of the list as a write handles it: its name, and either the text the file
// already holds for it, which is written back as it was, or the entry to encode for one the file
// does not hold yet.
type agentEntry struct {
	name  string
	raw   json.RawMessage
	fresh agentDocument
}

// agentList is the list a write starts from: its entries in order, and whether the file lays it out
// one entry per line.
type agentList struct {
	entries   []agentEntry
	multiline bool
}

func (l agentList) names() []string {
	names := make([]string, 0, len(l.entries))
	for _, entry := range l.entries {
		names = append(names, entry.name)
	}

	return names
}

// AddAgent puts a new agent into the list: last, or straight before or after the one named. A name
// already in the list is refused, and so is anything the settings module would not accept.
func (c *Config) AddAgent(dir string, agent NewAgent) (domain.Write, error) {
	project, list, err := c.startAgents(dir)
	if err != nil {
		return domain.Write{}, err
	}

	if slices.Contains(list.names(), agent.Name) {
		return domain.Write{}, fmt.Errorf("an agent named %q is already in the list; remove it first to change it", agent.Name)
	}

	at, err := domain.InsertAt(list.names(), agent.Before, agent.After)
	if err != nil {
		return domain.Write{}, err
	}

	entry := agentEntry{name: agent.Name, fresh: agentDocument{Name: agent.Name, Harness: agent.Harness, Model: agent.Model}}
	list.entries = slices.Insert(list.entries, at, entry)

	if err := c.writeAgents(dir, project, list); err != nil {
		return domain.Write{}, err
	}

	detail := fmt.Sprintf("added agent %s to %s", settings.DescribeAgent(settings.Agent{
		Name: agent.Name, Harness: agent.Harness, Model: agent.Model}), settingsName)

	if before, ok := agent.Before.Get(); ok {
		detail += ", before " + before
	} else if after, ok := agent.After.Get(); ok {
		detail += ", after " + after
	}

	return domain.Changed(detail), nil
}

// RemoveAgent takes an agent out of the list. It refuses while an order still names the agent, and
// says which, because removing it would leave that order pointing at nothing; and it refuses to
// remove the last agent, because an empty list means the default rather than no agents.
func (c *Config) RemoveAgent(dir, name string) (domain.Write, error) {
	project, list, err := c.startAgents(dir)
	if err != nil {
		return domain.Write{}, err
	}

	at := slices.Index(list.names(), name)
	if at < 0 {
		return domain.Write{}, domain.NotInList(name, list.names())
	}

	if referrers := domain.Referrers(project.doc, name); len(referrers) > 0 {
		return domain.Write{}, fmt.Errorf("agent %q is still named by %s; remove it there first",
			name, strings.Join(referrers, ", "))
	}

	if len(list.entries) == 1 {
		return domain.Write{}, fmt.Errorf(
			"agent %q is the only one in the list, and an empty list means the default (%s); add another agent first",
			name, settings.DescribeAgents(settings.DefaultAgents()))
	}

	list.entries = slices.Delete(list.entries, at, at+1)

	if err := c.writeAgents(dir, project, list); err != nil {
		return domain.Write{}, err
	}

	return domain.Changed(fmt.Sprintf("removed agent %s from %s", name, settingsName)), nil
}

// OrderAgents replaces the list's order. It refuses an order that does not name every agent exactly
// once, and writes nothing when the list is already in that order.
func (c *Config) OrderAgents(dir string, order []string) (domain.Write, error) {
	project, list, err := c.startAgents(dir)
	if err != nil {
		return domain.Write{}, err
	}

	if err := domain.CheckOrder(list.names(), order); err != nil {
		return domain.Write{}, err
	}

	if slices.Equal(list.names(), order) {
		return domain.Unchanged(fmt.Sprintf("%s already lists the agents in that order", settingsName)), nil
	}

	reordered := make([]agentEntry, 0, len(order))

	for _, name := range order {
		reordered = append(reordered, list.entries[slices.Index(list.names(), name)])
	}

	list.entries = reordered

	if err := c.writeAgents(dir, project, list); err != nil {
		return domain.Write{}, err
	}

	return domain.Changed(fmt.Sprintf("ordered the agents in %s: %s", settingsName, strings.Join(order, ", "))), nil
}

// startAgents is what every agents write starts from: settings that exist and that the settings
// module accepts, and the list they hold. Settings that list no agents start from the default, so a
// write keeps the agent every reader was already using.
func (c *Config) startAgents(dir string) (projectSettings, agentList, error) {
	project, err := c.readSettings(dir)
	if err != nil {
		return projectSettings{}, agentList{}, err
	}

	if err := validSettings(project.doc, settingsName+" is not valid, so nothing was changed"); err != nil {
		return projectSettings{}, agentList{}, err
	}

	raw, present, err := field(project.data, settings.FieldAgents)
	if err != nil {
		return projectSettings{}, agentList{}, fmt.Errorf("decode %s: %w", settingsName, err)
	}

	if !present || !listsAgents(project.doc) {
		defaults := settings.DefaultAgents()
		entries := make([]agentEntry, 0, len(defaults))

		for _, agent := range defaults {
			entries = append(entries, agentEntry{name: agent.Name, fresh: agentDocument{
				Name: agent.Name, Harness: agent.Harness, Model: agent.Model}})
		}

		return project, agentList{entries: entries, multiline: true}, nil
	}

	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return projectSettings{}, agentList{}, fmt.Errorf("decode %s: %w", settingsName, err)
	}

	// Validate accepted the list, so the names the module reads are the entries' own, in their order.
	names := settings.AgentNames(settings.Agents(project.doc))
	entries := make([]agentEntry, 0, len(items))

	for at, item := range items {
		entries = append(entries, agentEntry{name: names[at], raw: item})
	}

	return project, agentList{entries: entries, multiline: bytes.ContainsRune(raw, '\n')}, nil
}

// writeAgents writes the list back into the settings, changing only the list's own text, and only
// when the result is settings the module accepts. A result it would refuse leaves the file exactly as
// it was and says why in the module's words.
func (c *Config) writeAgents(dir string, project projectSettings, list agentList) error {
	body, err := withField(project.data, settings.FieldAgents, func(indent string) ([]byte, error) {
		return renderAgents(list, indent)
	})
	if err != nil {
		return fmt.Errorf("encode %s: %w", settingsName, err)
	}

	doc, err := decodeSettings(body)
	if err != nil {
		return err
	}

	if err := validSettings(doc, "that change would leave "+settingsName+" invalid, so nothing was changed"); err != nil {
		return err
	}

	if err := c.files.WriteFile(settingsPath(dir), body); err != nil {
		return fmt.Errorf("write %s: %w", settingsName, err)
	}

	return nil
}

// renderAgents lays the list out the way the file laid it out: one entry per line, one level in from
// the key, when the list spanned lines, and on one line otherwise. An entry the file already held is
// written back as its own text; a new one is encoded to match.
func renderAgents(list agentList, indent string) ([]byte, error) {
	inner := indent + "  "
	items := make([][]byte, 0, len(list.entries))

	for _, entry := range list.entries {
		if entry.raw != nil {
			items = append(items, entry.raw)

			continue
		}

		item, err := encode(entry.fresh, list.multiline, inner)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if !list.multiline {
		return slices.Concat([]byte("["), bytes.Join(items, []byte(", ")), []byte("]")), nil
	}

	return slices.Concat([]byte("[\n"+inner), bytes.Join(items, []byte(",\n"+inner)), []byte("\n"+indent+"]")), nil
}
