package config_test

import (
	"slices"
	"testing"

	"github.com/samber/do/v2"

	"github.com/lividlabs/codefall/cli/internal/config"
)

// A provider whose static return type is the concrete type compiles and then fails at runtime with
// "could not find service". Resolving the whole graph through a real injector is the only thing that
// catches that, so this test does exactly what the composition root does.
func TestRegisterProvidesEverythingCommandNeeds(t *testing.T) {
	injector := do.New()
	t.Cleanup(func() { _ = injector.Shutdown() })

	config.Register(injector)

	cmd := config.Command(injector)
	if cmd == nil {
		t.Fatal("Command returned nil")
	}

	if got := cmd.Name(); got != "config" {
		t.Errorf("Command().Name() = %q, want %q", got, "config")
	}

	var names []string
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}

	if want := []string{"agents", "harness", "persona", "review", "show", "skill-prefix"}; !slices.Equal(names, want) {
		t.Errorf("subcommands = %q, want %q", names, want)
	}
}
