package buildinfo

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"slices"
	"strconv"
	"testing"
)

const hash = "9bc737c41c85fb5a352b7685085d7acc7dac8e73"

// checkout is the build information the go command records for a binary built from a checkout.
func checkout(recorded string, modified bool) *debug.BuildInfo {
	return &debug.BuildInfo{
		Main: debug.Module{Path: "github.com/lividlabs/codefall", Version: recorded},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: hash},
			{Key: "vcs.modified", Value: strconv.FormatBool(modified)},
		},
	}
}

func TestDescribe(t *testing.T) {
	for _, tc := range []struct {
		name        string
		version     string
		commit      string
		info        *debug.BuildInfo
		wantVersion string
		wantDisplay string
	}{
		{
			name:    "a GoReleaser release",
			version: "0.14.0", commit: hash,
			info:        checkout("v0.14.0+dirty", true),
			wantVersion: "0.14.0",
			wantDisplay: "0.14.0 (9bc737c)",
		},
		{
			name:        "go install at a tag",
			info:        &debug.BuildInfo{Main: debug.Module{Version: "v0.14.0", Sum: "h1:abc="}},
			wantVersion: "0.14.0",
			wantDisplay: "0.14.0",
		},
		{
			name:        "a checkout at the tag",
			info:        checkout("v0.14.0", false),
			wantVersion: "0.14.0-dev (9bc737c)",
			wantDisplay: "0.14.0-dev (9bc737c)",
		},
		{
			name:        "a checkout at the tag, with uncommitted changes",
			info:        checkout("v0.14.0+dirty", true),
			wantVersion: "0.14.0-dev (9bc737c, uncommitted changes)",
			wantDisplay: "0.14.0-dev (9bc737c, uncommitted changes)",
		},
		{
			name:        "a checkout after the tag",
			info:        checkout("v0.14.1-0.20260921120000-9bc737c41c85", false),
			wantVersion: "0.14.0-dev (9bc737c)",
			wantDisplay: "0.14.0-dev (9bc737c)",
		},
		{
			name:        "a checkout after a prerelease tag",
			info:        checkout("v0.15.0-rc.1.0.20260921120000-9bc737c41c85+dirty", true),
			wantVersion: "0.15.0-rc.1-dev (9bc737c, uncommitted changes)",
			wantDisplay: "0.15.0-rc.1-dev (9bc737c, uncommitted changes)",
		},
		{
			name:        "a checkout with no tag below it",
			info:        checkout("v0.0.0-20260921120000-9bc737c41c85", false),
			wantVersion: "dev (9bc737c)",
			wantDisplay: "dev (9bc737c)",
		},
		{
			name:        "go run, which records no version and no commit",
			info:        &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			wantVersion: "dev",
			wantDisplay: "dev",
		},
		{
			name:        "no build information",
			wantVersion: "dev",
			wantDisplay: "dev",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := describe(tc.version, tc.commit, tc.info)

			if got := b.version(); got != tc.wantVersion {
				t.Errorf("version() = %q, want %q", got, tc.wantVersion)
			}

			if got := b.display(); got != tc.wantDisplay {
				t.Errorf("display() = %q, want %q", got, tc.wantDisplay)
			}
		})
	}
}

// goreleaserPath is the release configuration, relative to this package's directory, which is the
// working directory `go test` runs in.
const goreleaserPath = "../../../../.goreleaser.yml"

// The linker skips a -X flag whose package or variable does not exist, and says nothing, so a
// rename here or a typo there ships a release with no version. This holds the flags in
// .goreleaser.yml to this package's import path and to the variables declared above.
func TestGoReleaserStampsThisPackage(t *testing.T) {
	data, err := os.ReadFile(filepath.FromSlash(goreleaserPath))
	if err != nil {
		t.Fatalf("read %s: %v", goreleaserPath, err)
	}

	pkg := reflect.TypeFor[build]().PkgPath()
	flags := regexp.MustCompile(`-X\s+(\S+)\.(\w+)=`).FindAllStringSubmatch(string(data), -1)

	var stamped []string

	for _, flag := range flags {
		if flag[1] != pkg {
			t.Errorf("-X %s.%s names package %s, want %s", flag[1], flag[2], flag[1], pkg)
		}

		stamped = append(stamped, flag[2])
	}

	slices.Sort(stamped)

	// The variables in the var block at the top of buildinfo.go.
	if want := []string{"commit", "version"}; !slices.Equal(stamped, want) {
		t.Errorf(".goreleaser.yml stamps %v, want %v", stamped, want)
	}
}
