package validate

import (
	"regexp"
	"testing"
)

func check[V any](t *testing.T, rule func(V) error, good, bad V) {
	t.Helper()

	if err := rule(good); err != nil {
		t.Errorf("%v was refused: %v", good, err)
	}

	if rule(bad) == nil {
		t.Errorf("%v was accepted", bad)
	}
}

func TestRules(t *testing.T) {
	check(t, Required[string](), "x", "")
	check(t, Required[int](), 1, 0)
	check(t, MinLength(2), "ab", "a")
	check(t, MaxLength(2), "ää", "äää")
	check(t, Range(1, 3), 3, 4)
	check(t, OneOf("a", "b"), "b", "c")
	check(t, Matches(regexp.MustCompile(`^\d+$`)), "12", "a")
	check(t, Email, "a@example.com", "nope")
	check(t, All(MinLength(1), MaxLength(2)), "a", "abc")
}
