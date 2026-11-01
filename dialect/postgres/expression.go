package postgres

import "strings"

// typeWords continue a multi-word type name after a cast.
var typeWords = map[string]bool{
	"varying":   true,
	"precision": true,
	"without":   true,
	"with":      true,
	"time":      true,
	"zone":      true,
}

// SameExpression compares two expressions while ignoring casts, parentheses, and
// spacing, outside string literals. PostgreSQL prints one expression in different
// ways: a check on x IN ('a') reads back with its casts moved after a dump and
// restore, though it never changed. This runs only when exact text differs. It can
// miss a change that only regroups an expression, which is the better mistake here,
// because a false report would make every restored database look broken.
func (impl) SameExpression(a, b string) bool {
	return reduced(a) == reduced(b)
}

func reduced(expression string) string {
	var b strings.Builder

	for i := 0; i < len(expression); {
		c := expression[i]

		switch {
		case c == '\'':
			end := skipQuoted(expression, i, '\'', false)
			b.WriteString(expression[i:end])
			i = end
		case strings.HasPrefix(expression[i:], "::"):
			i = skipCast(expression, i+2)
		case c == '(' || c == ')' || c == ' ' || c == '\t' || c == '\n':
			i++
		default:
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}

			b.WriteByte(c)

			i++
		}
	}

	return b.String()
}

// skipCast returns the position after a type name starting at i, which can be
// several words, quoted, sized, or an array.
func skipCast(s string, i int) int {
	for {
		start := i

		if i < len(s) && s[i] == '"' {
			i = skipQuoted(s, i, '"', false)
		} else {
			for i < len(s) && (isWord(s[i]) || s[i] == '.') {
				i++
			}
		}

		if i == start {
			return i
		}

		if i < len(s) && s[i] == '(' {
			if end := strings.IndexByte(s[i:], ')'); end >= 0 {
				i += end + 1
			}
		}

		for strings.HasPrefix(s[i:], "[]") {
			i += 2
		}

		next := i
		for next < len(s) && s[next] == ' ' {
			next++
		}

		word := next
		for word < len(s) && isWord(s[word]) {
			word++
		}

		if !typeWords[strings.ToLower(s[next:word])] {
			return i
		}

		i = next
	}
}
