package dialect

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Bind prepares raw SQL for d. Three forms of argument are accepted:
//
//   - The dialect's own placeholders, such as $1, pass through unchanged.
//   - sql.Named arguments are written @name.
//   - Otherwise ? works as a placeholder, but only when the query uses no
//     numbered placeholders and has exactly one ? per argument. PostgreSQL also
//     has a ? operator on jsonb, and those queries fail one of these tests.
//
// With @name or ?, a slice directly inside IN (...) expands to one placeholder
// per element, and an empty slice becomes NULL. Anywhere else a slice is one
// value, so = ANY(@ids) passes an array. Text in strings, quoted identifiers,
// comments, and dollar quotes is left alone, as are operators like @>.
func Bind(d Dialect, query string, args []any) (string, []any, error) {
	named := map[string]any{}

	for _, a := range args {
		if n, ok := a.(sql.NamedArg); ok {
			named[n.Name] = n.Value
		}
	}

	switch {
	case len(named) > 0 && len(named) != len(args):
		return "", nil, errors.New("ant: raw query mixes named and positional arguments")
	case len(named) > 0:
		return bindNamed(d, query, named)
	case usesQuestionMarks(query, len(args)):
		return bindMarks(d, query, args)
	}

	return query, args, nil
}

func bindNamed(d Dialect, query string, named map[string]any) (string, []any, error) {
	b := &binder{d: d}
	used := map[string]string{}

	for i := 0; i < len(query); {
		if end := skipQuoted(query, i); end > i {
			b.out.WriteString(query[i:end])
			i = end

			continue
		}

		name := nameAt(query, i)
		if name == "" {
			b.out.WriteByte(query[i])
			i++

			continue
		}

		value, ok := named[name]
		if !ok {
			return "", nil, fmt.Errorf("ant: raw query has no argument named %q", name)
		}

		i += len(name) + 1

		if text, seen := used[name]; seen {
			b.out.WriteString(text)

			continue
		}

		used[name] = b.value(value, query[i:])
	}

	for name := range named {
		if _, ok := used[name]; !ok {
			return "", nil, fmt.Errorf("ant: raw query does not use argument %q", name)
		}
	}

	return b.out.String(), b.args, nil
}

func bindMarks(d Dialect, query string, args []any) (string, []any, error) {
	b := &binder{d: d}

	for i := 0; i < len(query); {
		if end := skipQuoted(query, i); end > i {
			b.out.WriteString(query[i:end])
			i = end

			continue
		}

		if query[i] != '?' {
			b.out.WriteByte(query[i])
			i++

			continue
		}

		i++
		b.value(args[0], query[i:])
		args = args[1:]
	}

	return b.out.String(), b.args, nil
}

// usesQuestionMarks reports whether ? marks are this query's placeholders.
func usesQuestionMarks(query string, args int) bool {
	if args == 0 {
		return false
	}

	marks := 0

	for i := 0; i < len(query); {
		if end := skipQuoted(query, i); end > i {
			i = end

			continue
		}

		switch {
		case query[i] == '?':
			marks++
		case query[i] == '$' && i+1 < len(query) && '0' <= query[i+1] && query[i+1] <= '9':
			return false
		}

		i++
	}

	return marks == args
}

// nameAt returns the parameter name of an @name starting at i, if there is one.
func nameAt(query string, i int) string {
	if query[i] != '@' {
		return ""
	}

	end := i + 1
	for end < len(query) && isNameByte(query[end], end == i+1) {
		end++
	}

	return query[i+1 : end]
}

// binder collects arguments and writes their placeholders.
type binder struct {
	d    Dialect
	out  strings.Builder
	args []any
}

// value writes the placeholders for v, given the query text after it, and
// returns what it wrote.
func (b *binder) value(v any, rest string) string {
	values := []any{v}

	if list, ok := expandable(v); ok && insideIn(b.out.String(), rest) {
		values = list
	}

	if len(values) == 0 {
		b.out.WriteString("NULL")

		return "NULL"
	}

	placeholders := make([]string, len(values))

	for i, value := range values {
		b.args = append(b.args, value)
		placeholders[i] = b.d.Placeholder(len(b.args))
	}

	text := strings.Join(placeholders, ", ")
	b.out.WriteString(text)

	return text
}

// expandable returns the elements of a slice that stands for a list of values.
// []byte is one value, as is any type that converts itself.
func expandable(v any) ([]any, bool) {
	if _, ok := v.(driver.Valuer); ok {
		return nil, false
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice || rv.Type().Elem().Kind() == reflect.Uint8 {
		return nil, false
	}

	list := make([]any, rv.Len())
	for i := range list {
		list[i] = rv.Index(i).Interface()
	}

	return list, true
}

// insideIn reports whether a placeholder between before and after is the whole
// of an IN (...) list.
func insideIn(before, after string) bool {
	before = strings.TrimSpace(before)
	if !strings.HasPrefix(strings.TrimSpace(after), ")") || !strings.HasSuffix(before, "(") {
		return false
	}

	before = strings.TrimSpace(strings.TrimSuffix(before, "("))

	return len(before) >= 2 && strings.EqualFold(before[len(before)-2:], "IN") &&
		(len(before) == 2 || !isNameByte(before[len(before)-3], true))
}

func isNameByte(c byte, first bool) bool {
	letter := c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'

	return letter || !first && '0' <= c && c <= '9'
}

// skipQuoted returns the end of the quoted region starting at i, or i if none
// starts there.
func skipQuoted(q string, i int) int {
	rest := q[i:]

	switch {
	case rest[0] == '\'', rest[0] == '"':
		for j := 1; j < len(rest); j++ {
			if rest[j] == rest[0] {
				if j+1 < len(rest) && rest[j+1] == rest[0] {
					j++

					continue
				}

				return i + j + 1
			}
		}

		return len(q)
	case strings.HasPrefix(rest, "--"):
		if end := strings.IndexByte(rest, '\n'); end >= 0 {
			return i + end
		}

		return len(q)
	case strings.HasPrefix(rest, "/*"):
		if end := strings.Index(rest, "*/"); end >= 0 {
			return i + end + 2
		}

		return len(q)
	case rest[0] == '$':
		tag := 1
		for tag < len(rest) && isNameByte(rest[tag], tag == 1) {
			tag++
		}

		if tag == len(rest) || rest[tag] != '$' {
			return i
		}

		delimiter := rest[:tag+1]
		if end := strings.Index(rest[tag+1:], delimiter); end >= 0 {
			return i + tag + 1 + end + len(delimiter)
		}

		return len(q)
	}

	return i
}
