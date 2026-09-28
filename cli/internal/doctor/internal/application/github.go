package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/text"
)

// ghAuthStatus is the shape of `gh auth status --json hosts`.
type ghAuthStatus struct {
	Hosts map[string][]ghAccount `json:"hosts"`
}

type ghAccount struct {
	State  string `json:"state"`
	Active bool   `json:"active"`
	Login  string `json:"login"`
	Scopes string `json:"scopes"`
	Error  string `json:"error"`
}

// githubHost is the only host doctor checks. GitHub Enterprise is out of scope.
const githubHost = "github.com"

// github runs checks 23 to 25.
//
// `gh auth status --json hosts` exits 0 even when authentication is broken, so its exit code is
// ignored and the answer is read out of the JSON instead.
func (d *Diagnose) github(ctx context.Context, dir string, results []domain.Result) []domain.Result {
	path, installed := d.runner.LookPath("gh").Get()
	if !installed {
		return append(results, domain.GHInstalled.Fail("gh is not on PATH", mo.Some("brew install gh")))
	}

	results = append(results, domain.GHInstalled.PassWithDetail(d.toolVersion(ctx, dir, path, "gh", "--version")))

	loginRemedy := mo.Some("gh auth login")

	result, err := d.runner.Run(ctx, dir, "gh", "auth", "status", "--json", "hosts")
	if err != nil {
		return append(results, domain.GHAuthenticated.Fail(
			fmt.Sprintf("Could not run gh auth status: %v", err), mo.None[string]()))
	}

	var status ghAuthStatus

	if err := json.Unmarshal([]byte(result.Stdout), &status); err != nil {
		detail := text.FirstLine(result.Stderr)
		if detail == "" {
			detail = text.FirstLine(result.Stdout)
		}

		return append(results, domain.GHAuthenticated.Fail(
			"Unexpected output from gh auth status: "+detail, loginRemedy))
	}

	account, found := activeAccount(status)
	if !found {
		return append(results, domain.GHAuthenticated.Fail("No active "+githubHost+" account", loginRemedy))
	}

	if account.State != "success" {
		detail := fmt.Sprintf("%s: state %q", account.Login, account.State)
		if account.Error != "" {
			detail += " " + account.Error
		}

		return append(results, domain.GHAuthenticated.Fail(detail, loginRemedy))
	}

	results = append(results, domain.GHAuthenticated.PassWithDetail("logged in as "+account.Login))

	return append(results, scopeResult(account.Scopes))
}

func activeAccount(status ghAuthStatus) (ghAccount, bool) {
	for _, account := range status.Hosts[githubHost] {
		if account.Active {
			return account, true
		}
	}

	return ghAccount{}, false
}

func scopeResult(scopes string) domain.Result {
	missing := domain.FindMissingScopes(parseScopes(scopes))
	if missing.IsEmpty() {
		return domain.GHScopes.Pass()
	}

	// Required scopes come first in both the detail and the remedy, so the one that blocks codefall
	// is the one read first.
	all := append(append([]string{}, missing.Required...), missing.Recommended...)

	remedy := "gh auth refresh"
	for _, scope := range all {
		remedy += " -s " + scope
	}

	detail := "The token is missing " + strings.Join(all, ", ")

	if len(missing.Required) > 0 {
		return domain.GHScopes.Fail(detail, mo.Some(remedy))
	}

	return domain.GHScopes.Warn(detail, mo.Some(remedy))
}

// parseScopes splits gh's single comma-separated scope string.
func parseScopes(scopes string) []string {
	var granted []string

	for _, scope := range strings.Split(scopes, ",") {
		if trimmed := strings.TrimSpace(scope); trimmed != "" {
			granted = append(granted, trimmed)
		}
	}

	return granted
}
