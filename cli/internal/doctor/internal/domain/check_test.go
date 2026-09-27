package domain

import (
	"testing"

	"github.com/samber/mo"
)

func TestStatusString(t *testing.T) {
	for _, tc := range []struct {
		status Status
		want   string
	}{
		{StatusPass, "PASS"},
		{StatusWarn, "WARN"},
		{StatusFail, "FAIL"},
		{Status(42), "UNKNOWN"},
	} {
		if got := tc.status.String(); got != tc.want {
			t.Errorf("Status(%d).String() = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestPassHasNoDetailAndNoRemedy(t *testing.T) {
	result := BeadsInstalled.Pass()

	if result.Status != StatusPass {
		t.Errorf("Status = %v, want PASS", result.Status)
	}

	if result.Detail.IsPresent() {
		t.Errorf("Detail = %v, want None", result.Detail)
	}

	if result.Remedy.IsPresent() {
		t.Errorf("Remedy = %v, want None", result.Remedy)
	}

	if result.Check != BeadsInstalled {
		t.Errorf("Check = %v, want %v", result.Check, BeadsInstalled)
	}
}

func TestPassWithDetailCarriesTheDetailOnly(t *testing.T) {
	result := BeadsInstalled.PassWithDetail("bd version 1.2.2")

	if got, ok := result.Detail.Get(); !ok || got != "bd version 1.2.2" {
		t.Errorf("Detail = %v, want Some(%q)", result.Detail, "bd version 1.2.2")
	}

	if result.Remedy.IsPresent() {
		t.Errorf("Remedy = %v, want None", result.Remedy)
	}
}

func TestWarnAndFailCarryDetailAndRemedy(t *testing.T) {
	warn := BeadsInitialized.Warn("no database", mo.Some("bd init"))
	if warn.Status != StatusWarn {
		t.Errorf("Warn status = %v, want WARN", warn.Status)
	}

	if got, ok := warn.Remedy.Get(); !ok || got != "bd init" {
		t.Errorf("Warn remedy = %v, want Some(%q)", warn.Remedy, "bd init")
	}

	fail := GHScopes.Fail("missing repo", mo.None[string]())
	if fail.Status != StatusFail {
		t.Errorf("Fail status = %v, want FAIL", fail.Status)
	}

	if got, ok := fail.Detail.Get(); !ok || got != "missing repo" {
		t.Errorf("Fail detail = %v, want Some(%q)", fail.Detail, "missing repo")
	}

	if fail.Remedy.IsPresent() {
		t.Errorf("Fail remedy = %v, want None", fail.Remedy)
	}
}

// allChecks is the twenty-one checks in the order doctor runs them.
var allChecks = []Check{
	CodefallDir, SettingsFile, SettingsJSON, SettingsComplete,
	ReviewsIgnored, TestsIgnored, StampIgnored, InteractionsMerged,
	HarnessNames, HarnessesInstalled, HarnessesLeftOver,
	LocalDeclared, LocalRunnable,
	TestDeclared, TestEquipped, TestDirExists,
	BeadsInstalled, BeadsInitialized, GHInstalled, GHAuthenticated, GHScopes,
}

func TestCheckIdentifiersAreDistinct(t *testing.T) {
	checks := allChecks

	seen := map[string]bool{}

	for _, check := range checks {
		if check.ID == "" || check.Title == "" {
			t.Errorf("check %+v has an empty field", check)
		}

		if seen[check.ID] {
			t.Errorf("duplicate check ID %q", check.ID)
		}

		seen[check.ID] = true
	}
}

func TestEveryCheckBelongsToOneOfTheSixCategories(t *testing.T) {
	want := map[string]Category{
		CodefallDir.ID:        CategorySettings,
		SettingsFile.ID:       CategorySettings,
		SettingsJSON.ID:       CategorySettings,
		SettingsComplete.ID:   CategorySettings,
		ReviewsIgnored.ID:     CategorySettings,
		TestsIgnored.ID:       CategorySettings,
		StampIgnored.ID:       CategorySettings,
		InteractionsMerged.ID: CategorySettings,
		HarnessNames.ID:       CategoryHarnesses,
		HarnessesInstalled.ID: CategoryHarnesses,
		HarnessesLeftOver.ID:  CategoryHarnesses,
		LocalDeclared.ID:      CategoryLocal,
		LocalRunnable.ID:      CategoryLocal,
		TestDeclared.ID:       CategoryTesting,
		TestEquipped.ID:       CategoryTesting,
		TestDirExists.ID:      CategoryTesting,
		BeadsInstalled.ID:     CategoryBeads,
		BeadsInitialized.ID:   CategoryBeads,
		GHInstalled.ID:        CategoryGitHub,
		GHAuthenticated.ID:    CategoryGitHub,
		GHScopes.ID:           CategoryGitHub,
	}

	for _, check := range allChecks {
		if got := check.Category; got != want[check.ID] {
			t.Errorf("%s category = %+v, want %+v", check.ID, got, want[check.ID])
		}
	}
}

func TestCategoriesAreDistinctAndNamed(t *testing.T) {
	categories := []Category{CategorySettings, CategoryHarnesses, CategoryLocal, CategoryTesting,
		CategoryBeads, CategoryGitHub}

	seen := map[string]bool{}

	for _, category := range categories {
		if category.ID == "" || category.Title == "" {
			t.Errorf("category %+v has an empty field", category)
		}

		if seen[category.ID] {
			t.Errorf("duplicate category ID %q", category.ID)
		}

		seen[category.ID] = true
	}
}
