package application

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// agentDocument is one agent of a list in the file's shape, for a list this command writes. The
// field order is the one init writes.
type agentDocument struct {
	Harness string            `json:"harness"`
	Model   mo.Option[string] `json:"model,omitzero"`
}

// entryItem is one entry of the agents list as a write handles it: which active agent it is for, and
// either the text the file holds for it, written back as it was, or the entry to encode for one the
// file does not hold yet. A write changes the one entry it is about and writes every other back as it
// was.
type entryItem struct {
	active string
	raw    json.RawMessage
	fresh  any
}

// entryList is the list a write starts from: its entries in order, and whether the file lays it out
// one entry per line.
type entryList struct {
	items     []entryItem
	multiline bool
}

func (l entryList) index(active string) int {
	return slices.IndexFunc(l.items, func(item entryItem) bool { return item.active == active })
}

// SetList sets one active agent's list for a feature to the agents given, in that order. An entry
// the file does not hold yet is added, last; one it holds keeps every other key it has. Only the
// list's own text changes, and a list already equal to the one given is left alone.
func (c *Config) SetList(dir, active, feature string, agents []settings.Agent) (domain.Write, error) {
	active, feature, err := parseTarget(active, feature)
	if err != nil {
		return domain.Write{}, err
	}

	if len(agents) == 0 {
		return domain.Write{}, domain.ErrEmptyList
	}

	project, list, err := c.startEntries(dir)
	if err != nil {
		return domain.Write{}, err
	}

	described := settings.DescribeAgents(agents)
	path := listPath(active, feature)

	if entry, ok := settings.EntryFor(project.doc, active).Get(); ok {
		if current, has := entry.List(feature).Get(); has && slices.Equal(current, agents) {
			return domain.Unchanged(fmt.Sprintf("%s already sets %s to %s", settingsName, path, described)), nil
		}
	}

	at := list.index(active)

	if at < 0 {
		list.items = append(list.items, entryItem{active: active, fresh: freshEntry(active, feature, agents)})
	} else {
		item, err := withList(list.items[at].raw, feature, agents)
		if err != nil {
			return domain.Write{}, fmt.Errorf("encode %s: %w", settingsName, err)
		}

		list.items[at].raw = item
	}

	if err := c.writeEntries(dir, project, list); err != nil {
		return domain.Write{}, err
	}

	return domain.Changed(fmt.Sprintf("set %s in %s: %s", path, settingsName, described)), nil
}

// ClearList removes one active agent's list for a feature, so a session in that harness uses the
// default entry's list again. An entry left with neither list is removed with it, since it would say
// nothing.
func (c *Config) ClearList(dir, active, feature string) (domain.Write, error) {
	active, feature, err := parseTarget(active, feature)
	if err != nil {
		return domain.Write{}, err
	}

	project, list, err := c.startEntries(dir)
	if err != nil {
		return domain.Write{}, err
	}

	path := listPath(active, feature)
	at := list.index(active)

	entry, has := settings.EntryFor(project.doc, active).Get()
	if at < 0 || !has || entry.List(feature).IsAbsent() {
		return domain.Unchanged(fmt.Sprintf("%s has no %s to clear", settingsName, path)), nil
	}

	detail := fmt.Sprintf("removed %s from %s", path, settingsName)

	if entry.List(otherFeature(feature)).IsAbsent() {
		list.items = slices.Delete(list.items, at, at+1)
		detail += ", and the " + active + " entry with it, since it named nothing else"
	} else {
		item, err := withoutField(list.items[at].raw, feature)
		if err != nil {
			return domain.Write{}, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		list.items[at].raw = item
	}

	if err := c.writeEntries(dir, project, list); err != nil {
		return domain.Write{}, err
	}

	return domain.Changed(detail), nil
}

// parseTarget checks the two words that name a list: an active agent the format knows, and a feature
// an entry holds a list for.
func parseTarget(active, feature string) (string, string, error) {
	active, err := settings.ParseActiveAgent(active)
	if err != nil {
		return "", "", err
	}

	feature, err = domain.ParseFeature(feature)
	if err != nil {
		return "", "", err
	}

	return active, feature, nil
}

// listPath is how a report names one list: the active agent and the feature, the words the command
// takes.
func listPath(active, feature string) string {
	return active + " " + feature
}

func otherFeature(feature string) string {
	if feature == settings.FeatureReview {
		return settings.FeatureConsult
	}

	return settings.FeatureReview
}

// startEntries is what every entries write starts from: settings that exist and that the settings
// module accepts, and the entries they hold, each as its own text. Settings that write no entries
// start from an empty list, since the default entry is what absence already means.
func (c *Config) startEntries(dir string) (projectSettings, entryList, error) {
	project, err := c.readSettings(dir)
	if err != nil {
		return projectSettings{}, entryList{}, err
	}

	if err := validSettings(project.doc, settingsName+" is not valid, so nothing was changed"); err != nil {
		return projectSettings{}, entryList{}, err
	}

	raw, present, err := field(project.data, settings.FieldAgents)
	if err != nil {
		return projectSettings{}, entryList{}, fmt.Errorf("decode %s: %w", settingsName, err)
	}

	if !present {
		return project, entryList{multiline: true}, nil
	}

	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return projectSettings{}, entryList{}, fmt.Errorf("decode %s: %w", settingsName, err)
	}

	list := entryList{items: make([]entryItem, 0, len(items)), multiline: bytes.ContainsRune(raw, '\n') || len(items) == 0}

	for _, item := range items {
		var fields struct {
			Active string `json:"activeAgent"`
		}

		// Validate accepted the list, so every item is an object naming an active agent.
		if err := json.Unmarshal(item, &fields); err != nil {
			return projectSettings{}, entryList{}, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		list.items = append(list.items, entryItem{active: fields.Active, raw: item})
	}

	return project, list, nil
}

// writeEntries writes the list back into the settings, changing only the list's own text, and only
// when the result is settings the module accepts. A result it would refuse leaves the file exactly as
// it was and says why in the module's words.
func (c *Config) writeEntries(dir string, project projectSettings, list entryList) error {
	body, err := withField(project.data, settings.FieldAgents, func(indent string) ([]byte, error) {
		return renderEntries(list, indent)
	})
	if err != nil {
		return fmt.Errorf("encode %s: %w", settingsName, err)
	}

	return c.writeSettings(dir, body)
}

// renderEntries lays the list out the way the file laid it out: one entry per line, one level in from
// the key, when the list spanned lines, and on one line otherwise. An entry read from the file is
// written as its own text, so a write to one entry never moves another; a fresh one is encoded to
// match, indented two spaces per level from the entries' own indentation.
func renderEntries(list entryList, indent string) ([]byte, error) {
	if len(list.items) == 0 {
		return []byte("[]"), nil
	}

	inner := indent + "  "
	items := make([][]byte, 0, len(list.items))

	for _, item := range list.items {
		if item.raw != nil {
			items = append(items, item.raw)

			continue
		}

		encoded, err := encode(item.fresh, list.multiline, inner)
		if err != nil {
			return nil, err
		}

		items = append(items, encoded)
	}

	if !list.multiline {
		return slices.Concat([]byte("["), bytes.Join(items, []byte(", ")), []byte("]")), nil
	}

	return slices.Concat([]byte("[\n"+inner), bytes.Join(items, []byte(",\n"+inner)), []byte("\n"+indent+"]")), nil
}

// freshEntry is an entry the file does not hold yet, in the file's shape: the active agent and the
// one list being set, in the key order init writes.
func freshEntry(active, feature string, agents []settings.Agent) any {
	documents := agentDocuments(agents)

	if feature == settings.FeatureReview {
		return struct {
			Active string          `json:"activeAgent"`
			Review []agentDocument `json:"review"`
		}{Active: active, Review: documents}
	}

	return struct {
		Active  string          `json:"activeAgent"`
		Consult []agentDocument `json:"consult"`
	}{Active: active, Consult: documents}
}

// withList is an entry's text with one feature's list set: replaced where the entry holds one, laid
// out the way the old one was, and added as the entry's last key where it does not, on one line.
func withList(item json.RawMessage, feature string, agents []settings.Agent) ([]byte, error) {
	entry, err := parseObject(item, 0, len(item))
	if err != nil {
		return nil, err
	}

	multiline := false
	if existing, held := entry.last(feature); held {
		multiline = bytes.ContainsRune(existing.raw, '\n')
	}

	return setMember(item, entry, feature, func(indent string) ([]byte, error) {
		return renderAgentList(agents, multiline, indent)
	})
}

// renderAgentList lays one list out on one line, or one agent per line one level in from the key when
// the list it replaces spanned lines.
func renderAgentList(agents []settings.Agent, multiline bool, indent string) ([]byte, error) {
	if !multiline {
		return encode(agentDocuments(agents), false, "")
	}

	inner := indent + "  "
	items := make([][]byte, 0, len(agents))

	for _, agent := range agents {
		item, err := encode(agentDocument{Harness: agent.Harness, Model: agent.Model}, true, inner)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return slices.Concat([]byte("[\n"+inner), bytes.Join(items, []byte(",\n"+inner)), []byte("\n"+indent+"]")), nil
}

func agentDocuments(agents []settings.Agent) []agentDocument {
	documents := make([]agentDocument, 0, len(agents))
	for _, agent := range agents {
		documents = append(documents, agentDocument{Harness: agent.Harness, Model: agent.Model})
	}

	return documents
}

// SetPosting turns posting review findings to a pull request on or off. The review block is created
// holding only the field when the settings have none, and a value already as asked is left alone.
func (c *Config) SetPosting(dir string, on bool) (domain.Write, error) {
	project, err := c.readSettings(dir)
	if err != nil {
		return domain.Write{}, err
	}

	if err := validSettings(project.doc, settingsName+" is not valid, so nothing was changed"); err != nil {
		return domain.Write{}, err
	}

	state := "off"
	if on {
		state = "on"
	}

	block, hasBlock := project.doc[settings.BlockReview].(map[string]any)
	if hasBlock {
		if value, present := block[settings.FieldPostToPullRequest].(bool); present && value == on {
			return domain.Unchanged(fmt.Sprintf("%s already has posting %s", settingsName, state)), nil
		}
	}

	render := func(string) ([]byte, error) { return encode(on, false, "") }

	var body []byte

	if hasBlock {
		body, err = withNestedField(project.data, settings.BlockReview, settings.FieldPostToPullRequest, render)
	} else {
		body, err = withField(project.data, settings.BlockReview, func(indent string) ([]byte, error) {
			return objectWithMember(indent, settings.FieldPostToPullRequest, render)
		})
	}

	if err != nil {
		return domain.Write{}, fmt.Errorf("encode %s: %w", settingsName, err)
	}

	if err := c.writeSettings(dir, body); err != nil {
		return domain.Write{}, err
	}

	return domain.Changed(fmt.Sprintf("set posting %s in %s", state, settingsName)), nil
}
