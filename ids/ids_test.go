package ids

import (
	"regexp"
	"sort"
	"testing"
	"time"
)

func TestUUID(t *testing.T) {
	shape := regexp.MustCompile(
		`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
	)

	first := UUID()
	if !shape.MatchString(first) {
		t.Fatalf("bad UUID %s", first)
	}

	time.Sleep(2 * time.Millisecond)

	if later := UUID(); later <= first {
		t.Fatalf("%s does not sort after %s", later, first)
	}
}

func TestULID(t *testing.T) {
	first := ULID()
	if len(first) != 26 {
		t.Fatalf("bad ULID %s", first)
	}

	time.Sleep(2 * time.Millisecond)

	if later := ULID(); later <= first {
		t.Fatalf("%s does not sort after %s", later, first)
	}
}

func TestSnowflake(t *testing.T) {
	if _, err := NewSnowflake(1024); err == nil {
		t.Fatal("node out of range was accepted")
	}

	s, err := NewSnowflake(7)
	if err != nil {
		t.Fatal(err)
	}

	keys := make([]int64, 10000)
	seen := map[int64]bool{}

	for i := range keys {
		keys[i] = s.Next()

		if seen[keys[i]] {
			t.Fatalf("duplicate key %d", keys[i])
		}

		seen[keys[i]] = true

		if keys[i]>>12&1023 != 7 {
			t.Fatalf("key %d lost its node", keys[i])
		}
	}

	if !sort.SliceIsSorted(keys, func(i, j int) bool { return keys[i] < keys[j] }) {
		t.Fatal("keys do not sort in order")
	}
}
