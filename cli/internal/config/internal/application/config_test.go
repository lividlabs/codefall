package application

import (
	"errors"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/userfile"
)

const workingDir = "/work"

var (
	settingsFull  = filepath.Join(workingDir, ".codefall", "settings.json")
	userFull      = filepath.Join(workingDir, ".codefall", "user.json")
	gitIgnoreFull = filepath.Join(workingDir, ".gitignore")
)

// initSettings is the file init writes for a Beads project, with one key init does not write, so a
// test can see it survive.
const initSettings = `{
  "$schema": "https://raw.githubusercontent.com/lividlabs/codefall-cli/main/schemas/settings.schema.json",
  "version": 1,
  "tracker": "beads",
  "harnesses": [
    "claude"
  ],
  "agents": [
    {
      "name": "subagent",
      "harness": "current"
    }
  ],
  "beads": {},
  "review": {
    "postToPullRequest": false
  },
  "note": "kept <as> written"
}
`

type fakeFileSystem struct {
	files  map[string][]byte
	errs   map[string]error
	writes []string
}

func newFakeFileSystem(files map[string]string) *fakeFileSystem {
	fake := &fakeFileSystem{files: map[string][]byte{}, errs: map[string]error{}}
	for path, body := range files {
		fake.files[path] = []byte(body)
	}

	return fake
}

func (f *fakeFileSystem) ReadFile(path string) ([]byte, error) {
	if err, ok := f.errs[path]; ok {
		return nil, err
	}

	data, ok := f.files[path]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}

	return data, nil
}

func (f *fakeFileSystem) WriteFile(path string, data []byte) error {
	if err, ok := f.errs["write "+path]; ok {
		return err
	}

	f.files[path] = data
	f.writes = append(f.writes, path)

	return nil
}

func project(settingsBody string) *fakeFileSystem {
	return newFakeFileSystem(map[string]string{settingsFull: settingsBody})
}

func added(name, runs string) NewAgent {
	return NewAgent{Name: name, Harness: runs, Model: mo.None[string](), Before: mo.None[string](), After: mo.None[string]()}
}

// --- agents --------------------------------------------------------------------------------------

// A new agent goes last by default, and only the list's own text changes: every other key keeps its
// place and its bytes, which a decode and re-encode would not have left alone.
func TestAddAgentAppendsAndLeavesTheRestOfTheFileAlone(t *testing.T) {
	files := project(initSettings)

	agent := added("architect", "codex")
	agent.Model = mo.Some("gpt-5-codex")

	write, err := NewConfig(files).AddAgent(workingDir, agent)
	if err != nil {
		t.Fatalf("AddAgent: %v", err)
	}

	if want := domain.Changed("added agent architect (codex, gpt-5-codex) to .codefall/settings.json"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	want := strings.Replace(initSettings, `      "harness": "current"
    }
  ],`, `      "harness": "current"
    },
    {
      "name": "architect",
      "harness": "codex",
      "model": "gpt-5-codex"
    }
  ],`, 1)
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

func TestAddAgentPlacesItBeforeOrAfterANamedAgent(t *testing.T) {
	files := project(initSettings)
	config := NewConfig(files)

	if _, err := config.AddAgent(workingDir, added("architect", "codex")); err != nil {
		t.Fatalf("AddAgent: %v", err)
	}

	second := added("second", "muse")
	second.Before = mo.Some("architect")

	write, err := config.AddAgent(workingDir, second)
	if err != nil {
		t.Fatalf("AddAgent --before: %v", err)
	}

	if !strings.HasSuffix(write.Detail, ", before architect") {
		t.Errorf("detail = %q, want it to say where the agent went", write.Detail)
	}

	third := added("third", "current")
	third.After = mo.Some("subagent")

	if _, err := config.AddAgent(workingDir, third); err != nil {
		t.Fatalf("AddAgent --after: %v", err)
	}

	agents, err := config.Agents(workingDir)
	if err != nil {
		t.Fatalf("Agents: %v", err)
	}

	if got, want := settings.AgentNames(agents), []string{"subagent", "third", "second", "architect"}; !slices.Equal(got, want) {
		t.Errorf("names = %q, want %q", got, want)
	}
}

// Settings that list no agents mean the default, so a write starts from it: the agent every reader
// was already using stays first, and the list is added to the file.
func TestAddAgentToSettingsThatListNoneKeepsTheDefault(t *testing.T) {
	files := project(`{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {}
}
`)

	if _, err := NewConfig(files).AddAgent(workingDir, added("architect", "codex")); err != nil {
		t.Fatalf("AddAgent: %v", err)
	}

	want := `{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "agents": [
    {
      "name": "subagent",
      "harness": "current"
    },
    {
      "name": "architect",
      "harness": "codex"
    }
  ]
}
`
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// A list written on one line stays on one line.
func TestAddAgentKeepsAOneLineList(t *testing.T) {
	files := project(`{"version": 1, "tracker": "beads", "harnesses": ["claude"], "beads": {},
  "agents": [{"name": "subagent", "harness": "current"}]}
`)

	if _, err := NewConfig(files).AddAgent(workingDir, added("architect", "codex")); err != nil {
		t.Fatalf("AddAgent: %v", err)
	}

	want := `"agents": [{"name": "subagent", "harness": "current"}, {"name":"architect","harness":"codex"}]}`
	if got := string(files.files[settingsFull]); !strings.Contains(got, want) {
		t.Errorf("settings.json =\n%s\nwant it to hold\n%s", got, want)
	}
}

// Each refusal leaves the file exactly as it was, and says why in words a person can act on — the
// settings module's own, where the module is what refused.
func TestAddAgentRefuses(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings string
		agent    NewAgent
		want     string
	}{
		{
			name:     "a name already in the list",
			settings: initSettings,
			agent:    added("subagent", "codex"),
			want:     `an agent named "subagent" is already in the list`,
		},
		{
			name:     "a name that is not a slug",
			settings: initSettings,
			agent:    added("Architect", "codex"),
			want:     "agents: [1].name: must match " + settings.AgentNamePattern,
		},
		{
			name:     "a harness codefall cannot start",
			settings: initSettings,
			agent:    added("architect", "claude-code"),
			want:     `agents: [1].harness: unknown value "claude-code"`,
		},
		{
			name:     "a place by an agent the list does not hold",
			settings: initSettings,
			agent:    NewAgent{Name: "architect", Harness: "codex", Before: mo.Some("reviewer")},
			want:     `no agent named "reviewer" in the list`,
		},
		{
			name:     "settings that are already invalid",
			settings: `{"version": 1, "harnesses": ["claude"]}`,
			agent:    added("architect", "codex"),
			want:     ".codefall/settings.json is not valid, so nothing was changed: tracker: missing",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := project(tc.settings)

			_, err := NewConfig(files).AddAgent(workingDir, tc.agent)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("AddAgent error = %v, want it to say %q", err, tc.want)
			}

			if len(files.writes) != 0 || string(files.files[settingsFull]) != tc.settings {
				t.Errorf("wrote %q, want the file left as it was", files.writes)
			}
		})
	}
}

// Every write refuses a directory init has not set up, and names the command that does.
func TestWritesRefuseADirectoryWithNoSettings(t *testing.T) {
	config := NewConfig(newFakeFileSystem(nil))

	_, addErr := config.AddAgent(workingDir, added("architect", "codex"))
	_, removeErr := config.RemoveAgent(workingDir, "subagent")
	_, orderErr := config.OrderAgents(workingDir, []string{"subagent"})
	_, personaErr := config.SetPersona(workingDir, userfile.PersonaProductManager)

	for _, err := range []error{addErr, removeErr, orderErr, personaErr} {
		if !errors.Is(err, errNotSetUp) || !strings.Contains(err.Error(), "run codefall init") {
			t.Errorf("error = %v, want it to name codefall init", err)
		}
	}
}

const referencedSettings = `{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "agents": [
    {"name": "subagent", "harness": "current"},
    {"name": "architect", "harness": "codex"},
    {"name": "second", "harness": "muse"}
  ],
  "review": {"postToPullRequest": false, "agents": ["architect", "subagent"]},
  "agentsByHarness": {"claude": ["architect"]}
}
`

func TestRemoveAgent(t *testing.T) {
	files := project(referencedSettings)

	write, err := NewConfig(files).RemoveAgent(workingDir, "second")
	if err != nil {
		t.Fatalf("RemoveAgent: %v", err)
	}

	if want := domain.Changed("removed agent second from .codefall/settings.json"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	want := strings.Replace(referencedSettings, `,
    {"name": "second", "harness": "muse"}`, "", 1)
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// Removing an agent an order still names would leave the order pointing at nothing, so it is refused
// and every order that names it is listed; so is removing the last one, which would mean the default.
func TestRemoveAgentRefuses(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings string
		agent    string
		want     string
	}{
		{
			name:     "an agent an order still names",
			settings: referencedSettings,
			agent:    "architect",
			want:     `agent "architect" is still named by review.agents, agentsByHarness.claude; remove it there first`,
		},
		{
			name:     "an agent the list does not hold",
			settings: referencedSettings,
			agent:    "reviewer",
			want:     `no agent named "reviewer" in the list (agents: subagent, architect, second)`,
		},
		{
			name:     "the last agent",
			settings: initSettings,
			agent:    "subagent",
			want:     `agent "subagent" is the only one in the list`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := project(tc.settings)

			_, err := NewConfig(files).RemoveAgent(workingDir, tc.agent)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("RemoveAgent error = %v, want it to say %q", err, tc.want)
			}

			if len(files.writes) != 0 {
				t.Errorf("wrote %q, want nothing written", files.writes)
			}
		})
	}
}

func TestOrderAgents(t *testing.T) {
	files := project(referencedSettings)
	config := NewConfig(files)

	write, err := config.OrderAgents(workingDir, []string{"second", "architect", "subagent"})
	if err != nil {
		t.Fatalf("OrderAgents: %v", err)
	}

	if want := "ordered the agents in .codefall/settings.json: second, architect, subagent"; write.Detail != want || !write.Changed {
		t.Errorf("write = %+v, want a change saying %q", write, want)
	}

	if got := string(files.files[settingsFull]); !strings.Contains(got, `"agents": [
    {"name": "second", "harness": "muse"},
    {"name": "architect", "harness": "codex"},
    {"name": "subagent", "harness": "current"}
  ],`) {
		t.Errorf("settings.json =\n%s\nwant the entries in the new order, each as it was written", got)
	}

	// The same order again writes nothing and says so.
	files.writes = nil

	write, err = config.OrderAgents(workingDir, []string{"second", "architect", "subagent"})
	if err != nil || write.Changed || len(files.writes) != 0 {
		t.Errorf("OrderAgents again = %+v, %v, wrote %q, want nothing changed", write, err, files.writes)
	}

	if _, err := config.OrderAgents(workingDir, []string{"second", "architect"}); err == nil ||
		!strings.Contains(err.Error(), "missing subagent") {
		t.Errorf("OrderAgents with one left out = %v, want it to name the one", err)
	}
}

// --- show ----------------------------------------------------------------------------------------

func TestShow(t *testing.T) {
	files := project(referencedSettings)
	files.files[userFull] = []byte(`{"version": 1, "persona": "product-manager"}`)

	shown, err := NewConfig(files).Show(workingDir)
	if err != nil {
		t.Fatalf("Show: %v", err)
	}

	if got, want := settings.AgentNames(shown.Agents), []string{"subagent", "architect", "second"}; !slices.Equal(got, want) || shown.Default {
		t.Errorf("Agents = %q, Default = %v, want %q and not the default", got, shown.Default, want)
	}

	if got, ok := shown.Review.Get(); !ok || !slices.Equal(got, []string{"architect", "subagent"}) {
		t.Errorf("Review = %v, want architect, subagent", shown.Review)
	}

	if shown.Consult.IsPresent() {
		t.Errorf("Consult = %v, want None", shown.Consult)
	}

	if want := map[string][]string{"claude": {"architect"}}; !maps.EqualFunc(shown.ByHarness, want, slices.Equal) {
		t.Errorf("ByHarness = %v, want %v", shown.ByHarness, want)
	}

	if want := (domain.Persona{Name: userfile.PersonaProductManager, FromFile: true}); shown.Persona != want {
		t.Errorf("Persona = %+v, want %+v", shown.Persona, want)
	}
}

func TestShowMarksTheDefaultList(t *testing.T) {
	shown, err := NewConfig(project(`{"version": 1}`)).Show(workingDir)
	if err != nil {
		t.Fatalf("Show: %v", err)
	}

	if !shown.Default || !slices.Equal(settings.AgentNames(shown.Agents), []string{settings.DefaultAgentName}) {
		t.Errorf("Show = %+v, want the default list, marked as the default", shown)
	}

	if want := (domain.Persona{Name: userfile.DefaultPersona}); shown.Persona != want {
		t.Errorf("Persona = %+v, want %+v", shown.Persona, want)
	}
}

// --- persona -------------------------------------------------------------------------------------

func TestPersonaReadsTheFileOrTheDefault(t *testing.T) {
	for _, tc := range []struct {
		name string
		file mo.Option[string]
		want domain.Persona
	}{
		{name: "no file", file: mo.None[string](), want: domain.Persona{Name: userfile.PersonaEngineer}},
		{name: "a file naming none", file: mo.Some(`{"version": 1}`), want: domain.Persona{Name: userfile.PersonaEngineer}},
		{
			name: "a file naming one",
			file: mo.Some(`{"version": 1, "persona": "product-manager"}`),
			want: domain.Persona{Name: userfile.PersonaProductManager, FromFile: true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := newFakeFileSystem(nil)
			if body, ok := tc.file.Get(); ok {
				files.files[userFull] = []byte(body)
			}

			got, err := NewConfig(files).Persona(workingDir)
			if err != nil || got != tc.want {
				t.Errorf("Persona = %+v, %v, want %+v, nil", got, err, tc.want)
			}
		})
	}

	files := newFakeFileSystem(map[string]string{userFull: `{"version": 1, "persona": "designer"}`})
	if _, err := NewConfig(files).Persona(workingDir); err == nil || !strings.Contains(err.Error(), `unknown value "designer"`) {
		t.Errorf("Persona of an invalid file = %v, want the module's reason", err)
	}
}

// A missing user file is created with the version and the persona, and .gitignore gains the line init
// writes, by init's own helper, before the file exists for git to see.
func TestSetPersonaCreatesTheFileAndIgnoresIt(t *testing.T) {
	files := project(initSettings)
	files.files[gitIgnoreFull] = []byte("node_modules/\n")

	writes, err := NewConfig(files).SetPersona(workingDir, userfile.PersonaProductManager)
	if err != nil {
		t.Fatalf("SetPersona: %v", err)
	}

	want := []domain.Write{
		domain.Changed("wrote .codefall/user.json (persona: product-manager)"),
		domain.Changed("added .codefall/user.json to .gitignore"),
	}
	if !slices.Equal(writes, want) {
		t.Errorf("writes = %+v, want %+v", writes, want)
	}

	if got, want := string(files.files[userFull]), "{\n  \"version\": 1,\n  \"persona\": \"product-manager\"\n}\n"; got != want {
		t.Errorf("user.json = %q, want %q", got, want)
	}

	wantIgnore, _ := settings.WithIgnoreLines(mo.Some("node_modules/\n"), []settings.IgnoreLine{
		{Entry: userfile.Name, Comment: userfile.GitIgnoreComment},
	})
	if got := string(files.files[gitIgnoreFull]); got != wantIgnore {
		t.Errorf(".gitignore = %q, want %q", got, wantIgnore)
	}

	if want := []string{gitIgnoreFull, userFull}; !slices.Equal(files.writes, want) {
		t.Errorf("wrote %q, want .gitignore before the user file", files.writes)
	}
}

// A user file that is there keeps every other key and its layout; only the persona changes, and the
// version is added when it has none. A .gitignore that already names the file is left alone.
func TestSetPersonaKeepsTheRestOfAnExistingFile(t *testing.T) {
	files := project(initSettings)
	files.files[gitIgnoreFull] = []byte(userfile.Name + "\n")
	files.files[userFull] = []byte("{\n  \"persona\": \"engineer\",\n  \"editor\": \"vim\"\n}\n")

	writes, err := NewConfig(files).SetPersona(workingDir, userfile.PersonaProductManager)
	if err != nil {
		t.Fatalf("SetPersona: %v", err)
	}

	if want := []domain.Write{domain.Changed("set persona to product-manager in .codefall/user.json")}; !slices.Equal(writes, want) {
		t.Errorf("writes = %+v, want %+v", writes, want)
	}

	want := "{\n  \"persona\": \"product-manager\",\n  \"editor\": \"vim\",\n  \"version\": 1\n}\n"
	if got := string(files.files[userFull]); got != want {
		t.Errorf("user.json = %q, want %q", got, want)
	}

	if slices.Contains(files.writes, gitIgnoreFull) {
		t.Error("wrote .gitignore, want a file that names the entry left alone")
	}
}

func TestSetPersonaThatIsAlreadySetWritesNothing(t *testing.T) {
	files := project(initSettings)
	files.files[gitIgnoreFull] = []byte(userfile.Name + "\n")
	files.files[userFull] = []byte(`{"version": 1, "persona": "product-manager"}`)

	writes, err := NewConfig(files).SetPersona(workingDir, userfile.PersonaProductManager)
	if err != nil {
		t.Fatalf("SetPersona: %v", err)
	}

	if want := []domain.Write{domain.Unchanged(".codefall/user.json already says persona: product-manager")}; !slices.Equal(writes, want) {
		t.Errorf("writes = %+v, want %+v", writes, want)
	}

	if len(files.writes) != 0 {
		t.Errorf("wrote %q, want nothing", files.writes)
	}
}

func TestSetPersonaRefuses(t *testing.T) {
	for _, tc := range []struct {
		name    string
		persona string
		user    string
		want    string
	}{
		{
			name:    "a persona codefall does not know",
			persona: "designer",
			want:    `unknown persona "designer" (known personas: engineer, product-manager)`,
		},
		{
			name:    "a user file the module would not accept",
			persona: userfile.PersonaEngineer,
			user:    `{"version": 2}`,
			want:    ".codefall/user.json is not valid, so nothing was changed: version: must be 1",
		},
		{
			name:    "a user file that is not an object",
			persona: userfile.PersonaEngineer,
			user:    `["engineer"]`,
			want:    "decode .codefall/user.json",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := project(initSettings)
			if tc.user != "" {
				files.files[userFull] = []byte(tc.user)
			}

			_, err := NewConfig(files).SetPersona(workingDir, tc.persona)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("SetPersona error = %v, want it to say %q", err, tc.want)
			}

			if len(files.writes) != 0 {
				t.Errorf("wrote %q, want nothing", files.writes)
			}
		})
	}
}
