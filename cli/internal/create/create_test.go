package create_test

import (
	"testing"

	"github.com/samber/do/v2"

	"github.com/lividlabs/codefall-cli/cli/internal/create"
	"github.com/lividlabs/codefall-cli/cli/internal/initcmd"
)

// A provider whose static return type is the concrete type compiles and then fails at runtime with
// "could not find service". Resolving the whole graph through a real injector is the only thing that
// catches that, so this test does exactly what the composition root does — with the real init
// command, so a rename of the flags create leaves out fails here too.
func TestRegisterProvidesEverythingCommandNeeds(t *testing.T) {
	injector := do.New()
	t.Cleanup(func() { _ = injector.Shutdown() })

	create.Register(injector)
	initcmd.Register(injector)

	initCommand := initcmd.Command(injector)
	for _, name := range []string{"location"} {
		if initCommand.Flags().Lookup(name) == nil {
			t.Errorf("init has no --%s, so create's list of flags to leave out is stale", name)
		}
	}

	cmd := create.Command(injector, initCommand)
	if got := cmd.Name(); got != "create" {
		t.Errorf("Command().Name() = %q, want %q", got, "create")
	}
}
