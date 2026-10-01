package main

// c1Filter works around charmbracelet/x issue #848: the x/ansi parser treats
// a raw 0x9C byte inside an OSC, DCS, APC, PM, or SOS string as the 8-bit
// String Terminator, even when it is a UTF-8 continuation byte. The rest of
// the string is then printed to the screen. Claude Code's window titles begin
// with U+2733 (E2 9C B3), so every title update leaked " Claude Code" into
// the pane at the cursor. Fix PRs #946 and #976 are open upstream; delete this
// file when one merges.
//
// The filter tracks string state across chunks and replaces any multi-byte
// character containing 0x9C inside a string with '?'. Bytes outside strings
// pass through untouched, as does everything inside a string that cannot be
// misread.
type c1Filter struct {
	inString bool
	escape   bool   // the previous byte was ESC
	pending  []byte // a multi-byte character being collected inside a string
	need     int    // continuation bytes still needed for pending

	// Because the filter sees every string payload before it is sanitised,
	// it also keeps the child's window title (OSC 0 or 2) with its original
	// characters. Title is the last one seen; TitleSeq counts the updates.
	Title    string
	TitleSeq int
	isOSC    bool
	payload  []byte
}

// endString closes the current string and records a title when it was one.
func (f *c1Filter) endString() {
	f.inString = false
	if f.isOSC {
		p := string(f.payload)
		if len(p) >= 2 && (p[0] == '0' || p[0] == '2') && p[1] == ';' {
			f.Title = p[2:]
			f.TitleSeq++
		}
	}
	f.isOSC = false
	f.payload = f.payload[:0]
}

func (f *c1Filter) Push(in []byte) []byte {
	out := make([]byte, 0, len(in))
	for _, b := range in {
		// Finish a character being collected inside a string.
		if f.need > 0 {
			if b&0xC0 == 0x80 {
				f.pending = append(f.pending, b)
				f.payload = append(f.payload, b)
				f.need--
				if f.need == 0 {
					out = append(out, f.flushPending()...)
				}
				continue
			}
			// Not a continuation byte: the character was malformed. Emit what
			// was held and fall through to handle b normally.
			out = append(out, f.flushPending()...)
			f.need = 0
		}

		if f.escape {
			f.escape = false
			if f.inString {
				// ESC \ ends the string; any other ESC inside a string is
				// passed through for the parser to judge.
				if b == '\\' {
					f.endString()
				}
				out = append(out, 0x1B, b)
				continue
			}
			switch b {
			case ']', 'P', '_', '^', 'X':
				f.inString = true
				f.isOSC = b == ']'
				f.payload = f.payload[:0]
			}
			out = append(out, 0x1B, b)
			continue
		}

		if b == 0x1B {
			f.escape = true
			continue
		}

		if !f.inString {
			out = append(out, b)
			continue
		}

		if b != 0x07 {
			f.payload = append(f.payload, b)
		}
		switch {
		case b == 0x07: // BEL ends an OSC
			f.endString()
			out = append(out, b)
		case b >= 0xC0: // lead byte of a multi-byte character
			f.pending = append(f.pending[:0], b)
			f.need = utf8Need(b)
			if f.need == 0 {
				out = append(out, b)
			}
		default:
			out = append(out, b)
		}
	}
	return out
}

// flushPending returns the collected character, or '?' when the parser would
// misread one of its bytes as a terminator.
func (f *c1Filter) flushPending() []byte {
	p := f.pending
	f.pending = f.pending[:0]
	for _, b := range p {
		if b == 0x9C {
			return []byte{'?'}
		}
	}
	return append([]byte(nil), p...)
}

func utf8Need(lead byte) int {
	switch {
	case lead&0xE0 == 0xC0:
		return 1
	case lead&0xF0 == 0xE0:
		return 2
	case lead&0xF8 == 0xF0:
		return 3
	}
	return 0
}
