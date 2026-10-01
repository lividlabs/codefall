package application

import (
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
	"github.com/lividlabs/codefall/cli/internal/shared/userfile"
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
  "$schema": "https://raw.githubusercontent.com/lividlabs/codefall/main/cli/schemas/settings.schema.json",
  "version": 1,
  "tracker": "beads",
  "harnesses": [
    "claude"
  ],
  "agents": [
    {
      "activeAgent": "default",
      "review": [
        {
          "harness": "current"
        }
      ],
      "consult": [
        {
          "harness": "current"
        }
      ]
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

// agent is one agent of a list as a test names it.
func agent(runs string) settings.Agent {
	return settings.Agent{Harness: runs, Model: mo.None[string]()}
}

// modelled is one agent on a chosen model.
func modelled(runs, model string) settings.Agent {
	return settings.Agent{Harness: runs, Model: mo.Some(model)}
}

// --- lists ---------------------------------------------------------------------------------------

// A list set on an entry the file holds replaces only that list's text: the entry's other list, and
// every other key in the file, keep their place and their bytes. The new list is laid out the way the
// old one was.
func TestSetListReplacesOneListAndLeavesTheRestOfTheFileAlone(t *testing.T) {
	files := project(initSettings)

	write, err := NewConfig(files).SetList(workingDir, "default", "review", []settings.Agent{agent("codex"), agent("current")})
	if err != nil {
		t.Fatalf("SetList: %v", err)
	}

	if want := domain.Changed("set default review in .codefall/settings.json: codex, current"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	want := strings.Replace(initSettings, `      "review": [
        {
          "harness": "current"
        }
      ],`, `      "review": [
        {
          "harness": "codex"
        },
        {
          "harness": "current"
        }
      ],`, 1)
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// An entry the file does not hold is added last, holding the active agent and the one list set, and
// laid out like the entries beside it.
func TestSetListAddsAnEntryForANewActiveAgent(t *testing.T) {
	files := project(initSettings)

	write, err := NewConfig(files).SetList(workingDir, "muse", "review", []settings.Agent{agent("claude"), modelled("codex", "gpt-5-codex")})
	if err != nil {
		t.Fatalf("SetList: %v", err)
	}

	if want := domain.Changed("set muse review in .codefall/settings.json: claude, codex:gpt-5-codex"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	want := strings.Replace(initSettings, `          "harness": "current"
        }
      ]
    }
  ],`, `          "harness": "current"
        }
      ]
    },
    {
      "activeAgent": "muse",
      "review": [
        {
          "harness": "claude"
        },
        {
          "harness": "codex",
          "model": "gpt-5-codex"
        }
      ]
    }
  ],`, 1)
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// A list added to an entry that lacks it goes in as the entry's last key, on one line, and the entry's
// existing list is untouched.
func TestSetListAddsTheOtherListToAnEntry(t *testing.T) {
	files := project(`{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "agents": [
    {
      "activeAgent": "muse",
      "review": [{ "harness": "claude" }]
    }
  ]
}
`)

	if _, err := NewConfig(files).SetList(workingDir, "muse", "consult", []settings.Agent{agent("codex")}); err != nil {
		t.Fatalf("SetList: %v", err)
	}

	want := `{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "agents": [
    {
      "activeAgent": "muse",
      "review": [{ "harness": "claude" }],
      "consult": [{"harness":"codex"}]
    }
  ]
}
`
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// Settings that write no list start from nothing: the entry set is the only one written, since the
// default entry is what absence already means.
func TestSetListOnSettingsThatWriteNoneAddsTheList(t *testing.T) {
	files := project(`{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {}
}
`)

	if _, err := NewConfig(files).SetList(workingDir, "codex", "consult", []settings.Agent{agent("claude"), agent("current")}); err != nil {
		t.Fatalf("SetList: %v", err)
	}

	want := `{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "agents": [
    {
      "activeAgent": "codex",
      "consult": [
        {
          "harness": "claude"
        },
        {
          "harness": "current"
        }
      ]
    }
  ]
}
`
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// A list written on one line stays on one line, and so does the list of entries.
func TestSetListKeepsAOneLineLayout(t *testing.T) {
	files := project(`{"version": 1, "tracker": "beads", "harnesses": ["claude"], "beads": {},
  "agents": [{"activeAgent": "default", "review": [{"harness": "current"}]}]}
`)

	if _, err := NewConfig(files).SetList(workingDir, "default", "review", []settings.Agent{agent("codex"), agent("current")}); err != nil {
		t.Fatalf("SetList: %v", err)
	}

	want := `"agents": [{"activeAgent": "default", "review": [{"harness":"codex"},{"harness":"current"}]}]}`
	if got := string(files.files[settingsFull]); !strings.Contains(got, want) {
		t.Errorf("settings.json =\n%s\nwant it to hold\n%s", got, want)
	}

	if _, err := NewConfig(files).SetList(workingDir, "muse", "consult", []settings.Agent{agent("claude")}); err != nil {
		t.Fatalf("SetList: %v", err)
	}

	want = `, {"activeAgent":"muse","consult":[{"harness":"claude"}]}]}`
	if got := string(files.files[settingsFull]); !strings.Contains(got, want) {
		t.Errorf("settings.json =\n%s\nwant it to hold\n%s", got, want)
	}
}

func TestSetListThatIsAlreadySetWritesNothing(t *testing.T) {
	files := project(initSettings)

	write, err := NewConfig(files).SetList(workingDir, "default", "consult", []settings.Agent{agent("current")})
	if err != nil {
		t.Fatalf("SetList: %v", err)
	}

	if want := domain.Unchanged(".codefall/settings.json already sets default consult to current"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	if len(files.writes) != 0 {
		t.Errorf("wrote %q, want nothing", files.writes)
	}
}

// Each refusal leaves the file exactly as it was, and says why in words a person can act on: the
// settings module's own, where the module is what refused.
func TestSetListRefuses(t *testing.T) {
	for _, tc := range []struct {
		name            string
		settings        string
		active, feature string
		agents          []settings.Agent
		want            string
	}{
		{
			name:     "an empty list",
			settings: initSettings,
			active:   "default", feature: "review",
			want: "name at least one agent, or clear the list",
		},
		{
			name:     "an active agent codefall does not know",
			settings: initSettings,
			active:   "cursor", feature: "review",
			agents: []settings.Agent{agent("current")},
			want:   `active agent "cursor" is not one codefall knows`,
		},
		{
			name:     "a feature an entry does not hold",
			settings: initSettings,
			active:   "default", feature: "testing",
			agents: []settings.Agent{agent("current")},
			want:   `"testing" is not a list an entry holds`,
		},
		{
			// The agent parsed, so it is the settings module that refuses the result.
			name:     "a harness codefall cannot start",
			settings: initSettings,
			active:   "default", feature: "review",
			agents: []settings.Agent{agent("cursor")},
			want:   `agents: [0].review[0].harness: unknown value "cursor"`,
		},
		{
			name:     "settings that are already invalid",
			settings: `{"version": 1, "harnesses": ["claude"]}`,
			active:   "default", feature: "review",
			agents: []settings.Agent{agent("current")},
			want:   ".codefall/settings.json is not valid, so nothing was changed: tracker: missing",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := project(tc.settings)

			_, err := NewConfig(files).SetList(workingDir, tc.active, tc.feature, tc.agents)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("SetList error = %v, want it to say %q", err, tc.want)
			}

			if len(files.writes) != 0 || string(files.files[settingsFull]) != tc.settings {
				t.Errorf("wrote %q, want the file left as it was", files.writes)
			}
		})
	}
}

const twoEntries = `{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "agents": [
    {"activeAgent": "default", "review": [{"harness": "current"}], "consult": [{"harness": "current"}]},
    {"activeAgent": "muse", "review": [{"harness": "claude"}], "consult": [{"harness": "codex"}]}
  ],
  "review": {"postToPullRequest": false}
}
`

// Clearing one list takes the key out of its entry and leaves the entry's other list, and every other
// entry, as they were.
func TestClearListRemovesTheKey(t *testing.T) {
	files := project(twoEntries)

	write, err := NewConfig(files).ClearList(workingDir, "muse", "consult")
	if err != nil {
		t.Fatalf("ClearList: %v", err)
	}

	if want := domain.Changed("removed muse consult from .codefall/settings.json"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	want := strings.Replace(twoEntries, `, "consult": [{"harness": "codex"}]}`, `}`, 1)
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// An entry left with neither list would say nothing, so it goes with its last list.
func TestClearListRemovesAnEntryLeftEmpty(t *testing.T) {
	files := project(twoEntries)
	config := NewConfig(files)

	if _, err := config.ClearList(workingDir, "muse", "consult"); err != nil {
		t.Fatalf("ClearList consult: %v", err)
	}

	write, err := config.ClearList(workingDir, "muse", "review")
	if err != nil {
		t.Fatalf("ClearList review: %v", err)
	}

	if want := domain.Changed("removed muse review from .codefall/settings.json, and the muse entry with it, since it named nothing else"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	want := strings.Replace(twoEntries, `,
    {"activeAgent": "muse", "review": [{"harness": "claude"}], "consult": [{"harness": "codex"}]}`, "", 1)
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

func TestClearListThatIsNotSetWritesNothing(t *testing.T) {
	files := project(twoEntries)

	write, err := NewConfig(files).ClearList(workingDir, "codex", "review")
	if err != nil {
		t.Fatalf("ClearList: %v", err)
	}

	if want := domain.Unchanged(".codefall/settings.json has no codex review to clear"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	if len(files.writes) != 0 {
		t.Errorf("wrote %q, want nothing", files.writes)
	}
}

// --- posting -------------------------------------------------------------------------------------

func TestSetPostingChangesTheFieldOrCreatesTheBlock(t *testing.T) {
	files := project(twoEntries)
	config := NewConfig(files)

	write, err := config.SetPosting(workingDir, true)
	if err != nil {
		t.Fatalf("SetPosting: %v", err)
	}

	if want := domain.Changed("set posting on in .codefall/settings.json"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	if got, want := string(files.files[settingsFull]), strings.Replace(twoEntries, `"postToPullRequest": false`, `"postToPullRequest": true`, 1); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}

	write, err = config.SetPosting(workingDir, true)
	if err != nil {
		t.Fatalf("SetPosting again: %v", err)
	}

	if want := domain.Unchanged(".codefall/settings.json already has posting on"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	noBlock := project(`{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {}
}
`)

	if _, err := NewConfig(noBlock).SetPosting(workingDir, true); err != nil {
		t.Fatalf("SetPosting with no block: %v", err)
	}

	want := `{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "review": {
    "postToPullRequest": true
  }
}
`
	if got := string(noBlock.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// Every write refuses a directory init has not set up, and names the command that does.
func TestWritesRefuseADirectoryWithNoSettings(t *testing.T) {
	config := NewConfig(newFakeFileSystem(nil))

	_, setErr := config.SetList(workingDir, "default", "review", []settings.Agent{agent("current")})
	_, clearErr := config.ClearList(workingDir, "default", "review")
	_, postErr := config.SetPosting(workingDir, true)
	_, personaErr := config.SetPersona(workingDir, userfile.PersonaProductManager)

	for _, err := range []error{setErr, clearErr, postErr, personaErr} {
		if !errors.Is(err, errNotSetUp) || !strings.Contains(err.Error(), "run codefall init") {
			t.Errorf("error = %v, want it to name codefall init", err)
		}
	}
}

// --- show ----------------------------------------------------------------------------------------

func TestShow(t *testing.T) {
	files := project(twoEntries)
	files.files[userFull] = []byte(`{"version": 1, "persona": "product-manager"}`)

	shown, err := NewConfig(files).Show(workingDir)
	if err != nil {
		t.Fatalf("Show: %v", err)
	}

	if got, want := settings.ActiveAgentNames(shown.Entries), []string{"default", "muse"}; !slices.Equal(got, want) || shown.Default {
		t.Errorf("Entries = %q, Default = %v, want %q and not the default", got, shown.Default, want)
	}

	if got, ok := shown.Entries[1].Consult.Get(); !ok || !slices.Equal(got, []settings.Agent{agent("codex")}) {
		t.Errorf("muse consult = %v, want codex", shown.Entries[1].Consult)
	}

	if shown.Posting {
		t.Error("Posting = true, want false")
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

	if !shown.Default || !slices.Equal(settings.ActiveAgentNames(shown.Entries), []string{settings.ActiveDefault}) {
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
