package config

import (
	"fmt"
	"strings"
)

// patchStringArray performs a text-level replacement of a TOML string-array
// value inside the [workspaces.<workspace>] section, preserving all comments,
// formatting, and surrounding content.  The result is always written as a
// multi-line array.
func patchStringArray(raw []byte, workspace, key string, values []string) ([]byte, error) {
	content := string(raw)

	// Normalise CRLF → LF so line splitting and matching work consistently.
	// The original line ending is restored when rejoining.
	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
		content = strings.ReplaceAll(content, "\r\n", "\n")
	}

	lines := strings.Split(content, "\n")

	sectionHeader := "[workspaces." + workspace + "]"

	sectionLine := -1 // index of the section header line
	keyStart := -1    // first line of the key = [...] entry
	keyEnd := -1      // last line of the key = [...] entry (inclusive)
	keyEndCol := -1   // column of the closing ']' on keyEnd

	inSection := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect table headers (single bracket only — skip [[ array-of-tables ]])
		if strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "[[") {
			rest := strings.TrimPrefix(trimmed, sectionHeader)
			if rest == "" || rest[0] == ' ' || rest[0] == '\t' || rest[0] == '#' {
				inSection = true
				sectionLine = i
				continue
			} else if inSection {
				// Entered the next section — stop searching
				break
			}
			continue
		}

		if !inSection {
			continue
		}

		// Skip blank lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check whether this line starts the target key
		if keyStart == -1 {
			k, _, _ := strings.Cut(trimmed, "=")
			if strings.TrimSpace(k) == key {
				keyStart = i
				endLine, endCol, err := findArrayEnd(lines, i)
				if err != nil {
					return nil, fmt.Errorf("find end of %q array in workspace %q: %w", key, workspace, err)
				}
				keyEnd = endLine
				keyEndCol = endCol
				break
			}
		}
	}

	// Extract the key line's own leading whitespace so we can preserve it.
	keyIndent := ""
	if keyStart >= 0 {
		orig := lines[keyStart]
		keyIndent = orig[:len(orig)-len(strings.TrimLeft(orig, " \t"))]
	}

	// Detect indentation from existing entries inside a multi-line array so we
	// can reproduce it. Fall back to key indentation + two spaces when the
	// array was single-line or empty (no entries to inspect).
	indent := keyIndent + "  "
	if keyStart >= 0 && keyEnd > keyStart {
		for i := keyStart + 1; i < keyEnd; i++ {
			entry := lines[i]
			trimmed := strings.TrimSpace(entry)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			// Measure leading whitespace of the first data line
			indent = entry[:len(entry)-len(strings.TrimLeft(entry, " \t"))]
			break
		}
	}

	// Capture any trailing content after the closing ']' (e.g. inline comments).
	trailing := ""
	if keyEnd >= 0 && keyEndCol >= 0 && keyEndCol+1 < len(lines[keyEnd]) {
		trailing = lines[keyEnd][keyEndCol+1:]
	}

	newBlock := buildMultiLineArray(key, keyIndent, indent, trailing, values)

	var result []string
	switch {
	case keyStart == -1 && sectionLine == -1:
		return nil, fmt.Errorf("workspace section %q not found in config", workspace)

	case keyStart == -1:
		// Key absent — insert after the section header line
		result = append(result, lines[:sectionLine+1]...)
		result = append(result, newBlock...)
		result = append(result, lines[sectionLine+1:]...)

	default:
		// Replace the existing key-value block
		result = append(result, lines[:keyStart]...)
		result = append(result, newBlock...)
		result = append(result, lines[keyEnd+1:]...)
	}

	return []byte(strings.Join(result, eol)), nil
}

// findArrayEnd returns the line index and column position of the closing ']'
// of the TOML array whose key-value line starts at startIdx.
// It handles single-line arrays, multi-line arrays, basic strings (with escape
// sequences), literal strings (single-quoted, no escaping), and inline comments.
func findArrayEnd(lines []string, startIdx int) (int, int, error) {
	depth := 0
	inString := false
	inLiteralString := false

	for i := startIdx; i < len(lines); i++ {
		s := lines[i]
		startJ := 0
		if i == startIdx {
			// Begin scanning after the '=' sign. This uses the first '='
			// on the line, which is correct for bare keys (paths, mounts)
			// but would break for quoted keys containing '='.
			if idx := strings.Index(s, "="); idx >= 0 {
				startJ = idx + 1
			}
		}

		j := startJ
	charLoop:
		for j < len(s) {
			ch := s[j]
			if inString {
				if ch == '\\' {
					j += 2 // skip the escaped character
					continue
				}
				if ch == '"' {
					inString = false
				}
			} else if inLiteralString {
				if ch == '\'' {
					inLiteralString = false
				}
			} else {
				switch ch {
				case '"':
					inString = true
				case '\'':
					inLiteralString = true
				case '[':
					depth++
				case ']':
					depth--
					if depth == 0 {
						return i, j, nil
					}
				case '#':
					break charLoop
				}
			}
			j++
		}
	}

	return -1, -1, fmt.Errorf("unterminated array")
}

// buildMultiLineArray returns the lines for a TOML multi-line array:
//
//	key = [
//	  "value1",
//	  "value2",
//	]
//
// keyIndent is the whitespace prefix for the key and closing bracket lines.
// indent is the whitespace prefix applied to each value line.
// trailing is any content to append after the closing ']' (e.g. inline comments).
func buildMultiLineArray(key, keyIndent, indent, trailing string, values []string) []string {
	lines := make([]string, 0, len(values)+2)
	lines = append(lines, keyIndent+key+" = [")
	for _, v := range values {
		lines = append(lines, indent+tomlQuoteString(v)+",")
	}
	lines = append(lines, keyIndent+"]"+trailing)
	return lines
}

// tomlQuoteString wraps s in a TOML basic string, escaping all characters that
// require it: backslash, double-quote, the named control sequences (\b, \f,
// \n, \r, \t), and any remaining C0 control characters via \uXXXX.
func tomlQuoteString(s string) string {
	var buf strings.Builder
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			buf.WriteString(`\\`)
		case '"':
			buf.WriteString(`\"`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&buf, `\u%04X`, r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
	return buf.String()
}
