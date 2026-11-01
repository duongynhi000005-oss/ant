package postgres

import "strings"

// Split separates a script into statements. It understands quotes, comments,
// dollar quoting, parentheses, and BEGIN ATOMIC bodies, so a semicolon inside any
// of them stays in its statement. Comments before a statement stay with it, and
// a piece holding only comments is dropped.
func (impl) Split(script string) []string {
	var out []string

	start, depth, atomic := 0, 0, 0
	last := ""

	flush := func(end int) {
		statement := strings.TrimSpace(script[start:end])
		if hasCode(statement) {
			out = append(out, statement)
		}
	}

	for i := 0; i < len(script); {
		c := script[i]

		switch {
		case c == '\'':
			escapes := i > 0 && (script[i-1] == 'E' || script[i-1] == 'e')
			i = skipQuoted(script, i, '\'', escapes)
		case c == '"':
			i = skipQuoted(script, i, '"', false)
		case strings.HasPrefix(script[i:], "--"):
			i = skipLine(script, i)
		case strings.HasPrefix(script[i:], "/*"):
			i = skipBlock(script, i)
		case c == '$':
			i = skipDollar(script, i)
		case c == '(':
			depth++
			i++
		case c == ')':
			depth--
			i++
		case c == ';' && depth == 0 && atomic == 0:
			flush(i)
			i++
			start = i
		case isWord(c):
			end := i
			for end < len(script) && isWord(script[end]) {
				end++
			}

			word := strings.ToUpper(script[i:end])

			switch {
			case word == "ATOMIC" && last == "BEGIN":
				atomic++
			case atomic > 0 && word == "CASE":
				atomic++
			case atomic > 0 && word == "END":
				atomic--
			}

			last = word
			i = end
		default:
			i++
		}
	}

	flush(len(script))

	return out
}

func isWord(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// skipQuoted returns the position after a quoted string starting at i. A doubled
// quote stays inside it, and so does a backslash escape in an E” string.
func skipQuoted(s string, i int, quote byte, escapes bool) int {
	for i++; i < len(s); i++ {
		switch {
		case escapes && s[i] == '\\':
			i++
		case s[i] == quote && i+1 < len(s) && s[i+1] == quote:
			i++
		case s[i] == quote:
			return i + 1
		}
	}

	return len(s)
}

func skipLine(s string, i int) int {
	if end := strings.IndexByte(s[i:], '\n'); end >= 0 {
		return i + end + 1
	}

	return len(s)
}

// skipBlock returns the position after a block comment, which can nest.
func skipBlock(s string, i int) int {
	depth := 0

	for i < len(s) {
		switch {
		case strings.HasPrefix(s[i:], "/*"):
			depth++
			i += 2
		case strings.HasPrefix(s[i:], "*/"):
			depth--
			i += 2

			if depth == 0 {
				return i
			}
		default:
			i++
		}
	}

	return len(s)
}

// skipDollar returns the position after a dollar-quoted string starting at i. A
// lone dollar, as in the parameter $1, is one character.
func skipDollar(s string, i int) int {
	end := i + 1
	for end < len(s) && isWord(s[end]) && !(end == i+1 && s[end] >= '0' && s[end] <= '9') {
		end++
	}

	if end >= len(s) || s[end] != '$' {
		return i + 1
	}

	tag := s[i : end+1]

	close := strings.Index(s[end+1:], tag)
	if close < 0 {
		return len(s)
	}

	return end + 1 + close + len(tag)
}

// hasCode reports whether statement holds anything but comments and space.
func hasCode(statement string) bool {
	for i := 0; i < len(statement); {
		switch {
		case strings.HasPrefix(statement[i:], "--"):
			i = skipLine(statement, i)
		case strings.HasPrefix(statement[i:], "/*"):
			i = skipBlock(statement, i)
		case strings.IndexByte(" \t\n\r", statement[i]) >= 0:
			i++
		default:
			return true
		}
	}

	return false
}
