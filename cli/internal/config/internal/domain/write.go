// Package domain holds what `codefall config` knows independently of any file: where a new agent goes
// in an order, whether a proposed order names every agent once, which orders still name an agent, and
// what a write did. The settings and user file formats themselves are the shared modules' (ADR-003).
package domain

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
