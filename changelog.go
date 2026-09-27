// Package changelog embeds the repository's CHANGELOG.md into the binary, so `codefall upgrade` can
// say which breaking changes lie between the version a project installed and the version running
// now, from the one record of them release-please writes (ADR-010). Like extensions/embed.go it sits
// where the file it embeds sits, because go:embed reaches no parent directory: the changelog is the
// repository's, release-please writes it at the root, and this is the one Go file the root holds.
package changelog

import _ "embed"

//go:embed CHANGELOG.md
var text []byte

// Text returns the changelog as the binary was built with it.
func Text() []byte {
	return text
}
