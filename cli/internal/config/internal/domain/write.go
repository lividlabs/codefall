// Package domain holds what `codefall config` knows independently of any file: the effective
// configuration a person reads, how a list of agents is read from what a person typed, and what a
// write did. The settings and user file formats themselves are the shared modules' (ADR-003).
package domain

import (
	"errors"
	"fmt"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// Write is what one write did to one file, and the sentence that says so. A write that found the file
// already saying what was asked leaves it alone and says that instead, so a script can run the same
// command twice.
type Write struct {
	Changed bool
	Detail  string
}

// Changed records a write that changed a file.
func Changed(detail string) Write {
	return Write{Changed: true, Detail: detail}
}

// Unchanged records a write that found nothing to change.
func Unchanged(detail string) Write {
	return Write{Changed: false, Detail: detail}
}

// ErrEmptyList refuses a list with no agents in it, which would leave a run nothing to try. The way to
// hand a feature back to the default entry is to clear the list.
var ErrEmptyList = errors.New("name at least one agent, or clear the list so the default entry's applies")

// ParseAgents reads a list of agents from the forms a person typed, `harness` or `harness:model`,
// in the order given. Every one has to parse, and the list has to hold at least one.
func ParseAgents(texts []string) ([]settings.Agent, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyList
	}

	agents := make([]settings.Agent, 0, len(texts))

	for _, text := range texts {
		agent, err := settings.ParseAgent(text)
		if err != nil {
			return nil, err
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

// ParseFeature returns the feature name when it is one an entry holds a list for, review or consult.
func ParseFeature(name string) (string, error) {
	for _, feature := range settings.Features() {
		if name == feature {
			return feature, nil
		}
	}

	return "", fmt.Errorf("%q is not a list an entry holds; name %s or %s", name, settings.FeatureReview, settings.FeatureConsult)
}
