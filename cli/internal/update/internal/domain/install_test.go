package domain

import (
	"errors"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/shared/version"
)

const (
	nativeBinary = "/Users/someone/.local/bin/codefall"
	miseBinary   = "/Users/someone/.local/share/mise/installs/packslip-github-com-lividlabs-codefall/0.26.0/.mise-bins/codefall"
)

func TestDetect(t *testing.T) {
	release := mo.Some(version.Version{Minor: 28})
	dev := mo.None[version.Version]()

	for _, tc := range []struct {
		name     string
		evidence Evidence
		want     Method
	}{
		{
			name:     "the receipt names the running binary",
			evidence: Evidence{Executable: nativeBinary, ReceiptPath: mo.Some(nativeBinary), Release: release},
			want:     MethodNative,
		},
		{
			name: "the receipt names the running binary by an unclean path",
			evidence: Evidence{
				Executable: nativeBinary, ReceiptPath: mo.Some("/Users/someone/.local/./bin/codefall"), Release: release,
			},
			want: MethodNative,
		},
		{
			name: "the receipt names another binary while mise's runs here",
			evidence: Evidence{
				Executable: miseBinary, ReceiptPath: mo.Some(nativeBinary), Release: release,
			},
			want: MethodMise,
		},
		{
			name:     "a binary under mise's default data directory",
			evidence: Evidence{Executable: miseBinary, Release: release},
			want:     MethodMise,
		},
		{
			name: "a binary under MISE_DATA_DIR's installs",
			evidence: Evidence{
				Executable:  "/opt/tools/installs/codefall/0.26.0/codefall",
				MiseDataDir: mo.Some("/opt/tools"),
				Release:     release,
			},
			want: MethodMise,
		},
		{
			name: "MISE_DATA_DIR is set and the binary is elsewhere",
			evidence: Evidence{
				Executable:  miseBinary,
				MiseDataDir: mo.Some("/opt/tools"),
				Release:     release,
			},
			want: MethodUnknown,
		},
		{
			name:     "a development build with no receipt",
			evidence: Evidence{Executable: "/Users/someone/go/bin/codefall", Release: dev},
			want:     MethodSource,
		},
		{
			name:     "a development build the receipt names is still native",
			evidence: Evidence{Executable: nativeBinary, ReceiptPath: mo.Some(nativeBinary), Release: dev},
			want:     MethodNative,
		},
		{
			name:     "a release binary nothing recognises",
			evidence: Evidence{Executable: nativeBinary, Release: release},
			want:     MethodUnknown,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Detect(tc.evidence); got != tc.want {
				t.Errorf("Detect() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestReceiptPath(t *testing.T) {
	if got, want := ReceiptPath(mo.None[string](), "/home/someone"),
		"/home/someone/.local/state/codefall/install.json"; got != want {
		t.Errorf("no XDG_STATE_HOME: %q, want %q", got, want)
	}

	if got, want := ReceiptPath(mo.Some(""), "/home/someone"),
		"/home/someone/.local/state/codefall/install.json"; got != want {
		t.Errorf("empty XDG_STATE_HOME: %q, want %q", got, want)
	}

	if got, want := ReceiptPath(mo.Some("/state"), "/home/someone"),
		"/state/codefall/install.json"; got != want {
		t.Errorf("XDG_STATE_HOME set: %q, want %q", got, want)
	}
}

// installScriptReceipt is a receipt exactly as scripts/install.sh writes it.
const installScriptReceipt = `{
  "path": "/Users/someone/.local/bin/codefall",
  "version": "0.28.0"
}
`

func TestReceiptRoundTrip(t *testing.T) {
	receipt, err := ParseReceipt([]byte(installScriptReceipt))
	if err != nil {
		t.Fatalf("ParseReceipt: %v", err)
	}

	if receipt != (Receipt{Path: nativeBinary, Version: "0.28.0"}) {
		t.Errorf("ParseReceipt = %+v", receipt)
	}

	if got := string(receipt.Encode()); got != installScriptReceipt {
		t.Errorf("Encode wrote\n%s\nwant what the install script writes\n%s", got, installScriptReceipt)
	}
}

func TestParseReceiptIgnoresUnknownFields(t *testing.T) {
	receipt, err := ParseReceipt([]byte(`{"path": "/bin/codefall", "version": "0.28.0", "later": true}`))
	if err != nil || receipt.Path != "/bin/codefall" {
		t.Errorf("ParseReceipt = %+v, %v", receipt, err)
	}
}

func TestParseReceiptRejects(t *testing.T) {
	for _, data := range []string{``, `{`, `{"version": "0.28.0"}`, `{"path": ""}`} {
		if _, err := ParseReceipt([]byte(data)); !errors.Is(err, ErrReceiptInvalid) {
			t.Errorf("ParseReceipt(%q) error = %v, want ErrReceiptInvalid", data, err)
		}
	}
}
