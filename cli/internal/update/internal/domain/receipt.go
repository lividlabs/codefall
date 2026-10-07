package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/samber/mo"
)

// ErrReceiptInvalid is a receipt that does not decode or names no binary.
var ErrReceiptInvalid = errors.New("install receipt is not valid")

// Receipt is what the install script records about the binary it installed, and what update records
// after it replaces that binary (ADR-015). A field neither side knows is ignored, so one can be added
// without breaking an older reader.
type Receipt struct {
	// Path is the binary, absolute, with symbolic links in its directory resolved.
	Path string `json:"path"`
	// Version is the release installed there.
	Version string `json:"version"`
}

// ReceiptPath is where the receipt lives: $XDG_STATE_HOME/codefall/install.json, and
// ~/.local/state/codefall/install.json when XDG_STATE_HOME is not set — on macOS as on Linux, because
// the install script writes it there.
func ReceiptPath(stateHome mo.Option[string], home string) string {
	dir, ok := stateHome.Get()
	if !ok || dir == "" {
		dir = filepath.Join(home, ".local", "state")
	}

	return filepath.Join(dir, "codefall", "install.json")
}

// ParseReceipt decodes a receipt.
func ParseReceipt(data []byte) (Receipt, error) {
	var receipt Receipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return Receipt{}, fmt.Errorf("%w: %w", ErrReceiptInvalid, err)
	}

	if receipt.Path == "" {
		return Receipt{}, fmt.Errorf("%w: it names no path", ErrReceiptInvalid)
	}

	return receipt, nil
}

// Encode is the receipt as the install script writes it: two-space indentation and a closing newline.
func (r Receipt) Encode() []byte {
	// A struct of two strings always encodes.
	data, _ := json.MarshalIndent(r, "", "  ")

	return append(data, '\n')
}
