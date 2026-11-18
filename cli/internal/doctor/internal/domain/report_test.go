package domain

import (
	"testing"

	"github.com/samber/mo"
)

func TestReportPreservesOrder(t *testing.T) {
	report := NewReport(
		CodefallDir.Pass(),
		SettingsFile.Fail("not found", mo.None[string]()),
		BeadsInitialized.Warn("no database", mo.Some("bd init")),
	)

	want := []string{CodefallDir.ID, SettingsFile.ID, BeadsInitialized.ID}

	results := report.Results()
	if len(results) != len(want) {
		t.Fatalf("Results() has %d entries, want %d", len(results), len(want))
	}

	for i, id := range want {
		if results[i].Check.ID != id {
			t.Errorf("Results()[%d].Check.ID = %q, want %q", i, results[i].Check.ID, id)
		}
	}
}

func TestReportSectionsGroupByCategoryInOrderOfFirstAppearance(t *testing.T) {
	report := NewReport(
		CodefallDir.Pass(),
		SettingsFile.Fail(".codefall/project.json not found", mo.None[string]()),
		BeadsInstalled.PassWithDetail("bd version 1.2.2"),
		BeadsInitialized.Warn("no database", mo.Some("bd init")),
		GHInstalled.PassWithDetail("gh version 2.97.0"),
	)

	sections := report.Sections()

	want := []struct {
		category Category
		ids      []string
		status   Status
	}{
		{CategorySettings, []string{CodefallDir.ID, SettingsFile.ID}, StatusFail},
		{CategoryBeads, []string{BeadsInstalled.ID, BeadsInitialized.ID}, StatusWarn},
		{CategoryGitHub, []string{GHInstalled.ID}, StatusPass},
	}

	if len(sections) != len(want) {
		t.Fatalf("Sections() has %d entries, want %d", len(sections), len(want))
	}

	for i, w := range want {
		section := sections[i]

		if section.Category != w.category {
			t.Errorf("Sections()[%d].Category = %+v, want %+v", i, section.Category, w.category)
		}

		if section.Status() != w.status {
			t.Errorf("%s status = %v, want %v", w.category.ID, section.Status(), w.status)
		}

		if len(section.Results) != len(w.ids) {
			t.Fatalf("%s has %d results, want %d", w.category.ID, len(section.Results), len(w.ids))
		}

		for j, id := range w.ids {
			if section.Results[j].Check.ID != id {
				t.Errorf("%s result %d = %q, want %q", w.category.ID, j, section.Results[j].Check.ID, id)
			}
		}
	}
}

func TestReportSectionsOfAnEmptyReport(t *testing.T) {
	if sections := NewReport().Sections(); len(sections) != 0 {
		t.Errorf("Sections() = %v, want none", sections)
	}
}

func TestSectionStatusIsTheWorstOfItsResults(t *testing.T) {
	for _, tc := range []struct {
		name    string
		results []Result
		want    Status
	}{
		{"no results", nil, StatusPass},
		{"all pass", []Result{GHInstalled.Pass(), GHScopes.Pass()}, StatusPass},
		{
			"a warning outranks a pass",
			[]Result{GHInstalled.Pass(), GHScopes.Warn("missing project", mo.None[string]())},
			StatusWarn,
		},
		{
			"a failure outranks a warning, whatever the order",
			[]Result{
				GHAuthenticated.Fail("No active github.com account", mo.None[string]()),
				GHScopes.Warn("missing project", mo.None[string]()),
			},
			StatusFail,
		},
	} {
		section := Section{Category: CategoryGitHub, Results: tc.results}
		if got := section.Status(); got != tc.want {
			t.Errorf("%s: Status() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestReportSectionsWithIssuesCountsCategoriesNotChecks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		report Report
		want   int
	}{
		{"empty", NewReport(), 0},
		{"all pass", NewReport(CodefallDir.Pass(), BeadsInstalled.Pass()), 0},
		{
			"two problems in one category count once",
			NewReport(
				CodefallDir.Pass(),
				SettingsFile.Fail("not found", mo.None[string]()),
				SettingsJSON.Warn("x", mo.None[string]()),
			),
			1,
		},
		{
			"a warning counts as an issue",
			NewReport(
				SettingsFile.Fail("not found", mo.None[string]()),
				BeadsInitialized.Warn("no database", mo.Some("bd init")),
				GHInstalled.Pass(),
			),
			2,
		},
	} {
		if got := tc.report.SectionsWithIssues(); got != tc.want {
			t.Errorf("%s: SectionsWithIssues() = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestReportFailedCountsOnlyFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		report Report
		want   int
	}{
		{"empty", NewReport(), 0},
		{"all pass", NewReport(CodefallDir.Pass(), SettingsFile.Pass()), 0},
		{"warnings do not count", NewReport(BeadsInitialized.Warn("x", mo.None[string]())), 0},
		{
			"two failures",
			NewReport(
				CodefallDir.Pass(),
				SettingsFile.Fail("not found", mo.None[string]()),
				BeadsInitialized.Warn("x", mo.None[string]()),
				GHScopes.Fail("missing repo", mo.None[string]()),
			),
			2,
		},
	} {
		if got := tc.report.Failed(); got != tc.want {
			t.Errorf("%s: Failed() = %d, want %d", tc.name, got, tc.want)
		}
	}
}
