package application

import (
	"reflect"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

// withBlocks is a file holding two blocks, Codex through a provider and a variant that calls it
// directly, beside keys a write must leave alone.
const withBlocks = `{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "harnessConfig": {
    "codex": {
      "modelFlag": "--model",
      "provider": "amazon-bedrock-runtime",
      "args": ["-c", "model_reasoning_effort=high"],
      "env": "aws configure export-credentials --format env"
    },
    "codex-direct": { "harness": "codex", "args": ["-c", "model_reasoning_effort=high"] }
  },
  "agents": [
    {"activeAgent": "claude", "review": [{"harness": "codex-direct", "model": "gpt-6-astra"}, {"harness": "current"}]}
  ],
  "review": {"postToPullRequest": false}
}
`

func codexViaBedrock() settings.HarnessConfig {
	return settings.HarnessConfig{
		Harness:   "codex",
		ModelFlag: mo.Some("--model"),
		Provider:  mo.Some("amazon-bedrock-runtime"),
		Env:       mo.Some("aws configure export-credentials --format env"),
		Args:      []string{"-c", "model_reasoning_effort=high"},
	}
}

// --- harness blocks ------------------------------------------------------------------------------

// A file with no harnessConfig gains the object holding the one block, laid out on lines, after the
// file's last key; nothing else moves.
func TestSetHarnessConfigCreatesTheObject(t *testing.T) {
	files := project(initSettings)

	write, err := NewConfig(files).SetHarnessConfig(workingDir, "codex", settings.HarnessConfig{
		ModelFlag: mo.Some("--model"),
		Provider:  mo.Some("amazon-bedrock-runtime"),
		Args:      []string{"-c", "model_reasoning_effort=high"},
		Env:       mo.Some("aws configure export-credentials --format env"),
	})
	if err != nil {
		t.Fatalf("SetHarnessConfig: %v", err)
	}

	if want := domain.Changed("set harnessConfig.codex in .codefall/settings.json"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	want := strings.Replace(initSettings, `  "note": "kept <as> written"
}`, `  "note": "kept <as> written",
  "harnessConfig": {
    "codex": {
      "modelFlag": "--model",
      "provider": "amazon-bedrock-runtime",
      "args": [
        "-c",
        "model_reasoning_effort=high"
      ],
      "env": "aws configure export-credentials --format env"
    }
  }
}`, 1)
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// A block that is there is replaced with exactly the fields given, in its own layout, and the
// other block and every other key keep their bytes.
func TestSetHarnessConfigReplacesOneBlockAndLeavesTheRestAlone(t *testing.T) {
	files := project(withBlocks)

	if _, err := NewConfig(files).SetHarnessConfig(workingDir, "codex-direct", settings.HarnessConfig{
		Harness: "codex", ModelFlag: mo.Some("-m"),
	}); err != nil {
		t.Fatalf("SetHarnessConfig: %v", err)
	}

	want := strings.Replace(withBlocks,
		`"codex-direct": { "harness": "codex", "args": ["-c", "model_reasoning_effort=high"] }`,
		`"codex-direct": {"harness":"codex","modelFlag":"-m"}`, 1)
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// A new block joins an object that is there, laid out like the object, and a variant's harness is
// written where a harness-named key's is not.
func TestSetHarnessConfigAddsABlockToTheObject(t *testing.T) {
	files := project(withBlocks)

	if _, err := NewConfig(files).SetHarnessConfig(workingDir, "muse-fast", settings.HarnessConfig{
		Harness: "muse", Args: []string{"--fast"},
	}); err != nil {
		t.Fatalf("SetHarnessConfig: %v", err)
	}

	want := strings.Replace(withBlocks,
		`"codex-direct": { "harness": "codex", "args": ["-c", "model_reasoning_effort=high"] }
  },`, `"codex-direct": { "harness": "codex", "args": ["-c", "model_reasoning_effort=high"] },
    "muse-fast": {
      "harness": "muse",
      "args": [
        "--fast"
      ]
    }
  },`, 1)
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}

	// A block under a harness name carries no harness field: the key says.
	if _, err := NewConfig(files).SetHarnessConfig(workingDir, "claude", settings.HarnessConfig{
		Provider: mo.Some("bedrock"),
	}); err != nil {
		t.Fatalf("SetHarnessConfig claude: %v", err)
	}

	if got := string(files.files[settingsFull]); !strings.Contains(got, "\"claude\": {\n      \"provider\": \"bedrock\"\n    }") {
		t.Errorf("settings.json =\n%s\nwant a claude block holding only the provider", got)
	}
}

func TestSetHarnessConfigThatIsAlreadySetWritesNothing(t *testing.T) {
	files := project(withBlocks)

	block := codexViaBedrock()
	block.Harness = ""

	write, err := NewConfig(files).SetHarnessConfig(workingDir, "codex", block)
	if err != nil {
		t.Fatalf("SetHarnessConfig: %v", err)
	}

	if want := domain.Unchanged(".codefall/settings.json already sets harnessConfig.codex that way"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	if len(files.writes) != 0 {
		t.Errorf("wrote %q, want nothing", files.writes)
	}
}

// Each refusal leaves the file exactly as it was, and says why in words a person can act on: the
// settings module's own, where the module is what refused.
func TestSetHarnessConfigRefuses(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings string
		key      string
		block    settings.HarnessConfig
		want     string
	}{
		{
			name:     "an empty block",
			settings: withBlocks,
			key:      "codex",
			want:     "give the block at least one field, or clear it",
		},
		{
			name:     "a harness on a harness-named key that is not the key",
			settings: withBlocks,
			key:      "codex",
			block:    settings.HarnessConfig{Harness: "muse"},
			want:     `harnessConfig.codex.harness: not allowed; the key "codex" already names the harness`,
		},
		{
			name:     "a variant without a harness",
			settings: withBlocks,
			key:      "bedrock",
			block:    settings.HarnessConfig{Provider: mo.Some("amazon-bedrock-runtime")},
			want:     "that change would leave .codefall/settings.json invalid, so nothing was changed: harnessConfig.bedrock.harness: missing",
		},
		{
			name:     "a variant on current",
			settings: withBlocks,
			key:      "mine",
			block:    settings.HarnessConfig{Harness: "current"},
			want:     `harnessConfig.mine.harness: unknown value "current"`,
		},
		{
			name:     "a key that is not a slug",
			settings: withBlocks,
			key:      "Codex Direct",
			block:    settings.HarnessConfig{Harness: "codex"},
			want:     "harnessConfig.Codex Direct: the key must be lowercase letters and digits joined by hyphens",
		},
		{
			name:     "settings that are already invalid",
			settings: `{"version": 1, "harnesses": ["claude"]}`,
			key:      "codex",
			block:    settings.HarnessConfig{Provider: mo.Some("x")},
			want:     ".codefall/settings.json is not valid, so nothing was changed: tracker: missing",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := project(tc.settings)

			_, err := NewConfig(files).SetHarnessConfig(workingDir, tc.key, tc.block)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("SetHarnessConfig error = %v, want it to say %q", err, tc.want)
			}

			if len(files.writes) != 0 || string(files.files[settingsFull]) != tc.settings {
				t.Errorf("wrote %q, want the file left as it was", files.writes)
			}
		})
	}
}

// Clearing one block takes it out and leaves the other, and every other key, as they were. A block a
// list names is refused by the settings module, since the list would then name nothing.
func TestClearHarnessConfigRemovesTheBlock(t *testing.T) {
	files := project(withBlocks)

	write, err := NewConfig(files).ClearHarnessConfig(workingDir, "codex")
	if err != nil {
		t.Fatalf("ClearHarnessConfig: %v", err)
	}

	if want := domain.Changed("removed harnessConfig.codex from .codefall/settings.json"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	want := strings.Replace(withBlocks, `    "codex": {
      "modelFlag": "--model",
      "provider": "amazon-bedrock-runtime",
      "args": ["-c", "model_reasoning_effort=high"],
      "env": "aws configure export-credentials --format env"
    },
`, "", 1)
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}

	_, err = NewConfig(files).ClearHarnessConfig(workingDir, "codex-direct")
	if err == nil || !strings.Contains(err.Error(), `agents: [0].review[0].harness: unknown value "codex-direct"`) {
		t.Errorf("clearing a block a list names: error = %v, want the settings module's refusal", err)
	}

	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json changed on a refusal:\n%s", got)
	}
}

// The object goes with its last block, so a file that says nothing about any harness carries no
// empty object saying so.
func TestClearHarnessConfigRemovesTheObjectLeftEmpty(t *testing.T) {
	files := project(`{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "harnessConfig": {
    "codex": { "provider": "amazon-bedrock-runtime" }
  },
  "review": {"postToPullRequest": false}
}
`)

	write, err := NewConfig(files).ClearHarnessConfig(workingDir, "codex")
	if err != nil {
		t.Fatalf("ClearHarnessConfig: %v", err)
	}

	if want := domain.Changed("removed harnessConfig.codex from .codefall/settings.json, and harnessConfig with it, since it held nothing else"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	want := `{
  "version": 1,
  "tracker": "beads",
  "harnesses": ["claude"],
  "beads": {},
  "review": {"postToPullRequest": false}
}
`
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

func TestClearHarnessConfigThatIsNotSetWritesNothing(t *testing.T) {
	for _, body := range []string{withBlocks, initSettings} {
		files := project(body)

		write, err := NewConfig(files).ClearHarnessConfig(workingDir, "muse")
		if err != nil {
			t.Fatalf("ClearHarnessConfig: %v", err)
		}

		if want := domain.Unchanged(".codefall/settings.json has no harnessConfig.muse to clear"); write != want {
			t.Errorf("write = %+v, want %+v", write, want)
		}

		if len(files.writes) != 0 {
			t.Errorf("wrote %q, want nothing", files.writes)
		}
	}
}

func TestHarnessConfigsReadsTheBlocks(t *testing.T) {
	configs, err := NewConfig(project(withBlocks)).HarnessConfigs(workingDir)
	if err != nil {
		t.Fatalf("HarnessConfigs: %v", err)
	}

	want := map[string]settings.HarnessConfig{
		"codex": codexViaBedrock(),
		"codex-direct": {
			Harness: "codex", ModelFlag: mo.None[string](), Provider: mo.None[string](), Env: mo.None[string](),
			Args: []string{"-c", "model_reasoning_effort=high"},
		},
	}
	if !reflect.DeepEqual(configs, want) {
		t.Errorf("HarnessConfigs = %+v, want %+v", configs, want)
	}

	shown, err := NewConfig(project(withBlocks)).Show(workingDir)
	if err != nil {
		t.Fatalf("Show: %v", err)
	}

	if !reflect.DeepEqual(shown.HarnessConfigs, want) {
		t.Errorf("Show().HarnessConfigs = %+v, want %+v", shown.HarnessConfigs, want)
	}

	none, err := NewConfig(project(initSettings)).HarnessConfigs(workingDir)
	if err != nil || len(none) != 0 {
		t.Errorf("HarnessConfigs of settings with none = %+v, %v; want none", none, err)
	}
}

// A list may name a variant by its key, and the write is refused, in the settings module's words,
// when the file holds no block under that key.
func TestSetListAcceptsAVariantKeyOnlyWhenTheFileHoldsIt(t *testing.T) {
	files := project(withBlocks)

	write, err := NewConfig(files).SetList(workingDir, "claude", "consult", []settings.Agent{modelled("codex-direct", "gpt-6-astra")})
	if err != nil {
		t.Fatalf("SetList: %v", err)
	}

	if want := domain.Changed("set claude consult in .codefall/settings.json: codex-direct:gpt-6-astra"); write != want {
		t.Errorf("write = %+v, want %+v", write, want)
	}

	_, err = NewConfig(project(initSettings)).SetList(workingDir, "claude", "consult", []settings.Agent{agent("codex-direct")})
	if err == nil || !strings.Contains(err.Error(), `that change would leave .codefall/settings.json invalid, so nothing was changed: `+
		`agents: [1].consult[0].harness: unknown value "codex-direct" (expected "agy", "claude", "codex", "current", "muse", "opencode", or a harnessConfig key)`) {
		t.Errorf("SetList with a key the file lacks: error = %v, want the settings module's refusal", err)
	}
}
