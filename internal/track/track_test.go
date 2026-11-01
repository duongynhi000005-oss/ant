package track

import (
	"testing"
	"time"
	"unsafe"

	"github.com/skudra-go/ant/internal/meta"
)

type address struct {
	City string
	Tags []string
}

type row struct {
	ID    int64
	Name  string
	Bio   *string
	Data  []byte
	At    time.Time
	Tags  []string
	Home  address
	Attrs map[string]string `ant:"column=attrs"`
	Peers []row
}

func fresh() *row {
	return &row{
		ID:    1,
		Name:  "a",
		Bio:   new("b"),
		Data:  []byte("c"),
		Tags:  []string{"d"},
		At:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Home:  address{City: "e", Tags: []string{"f"}},
		Attrs: map[string]string{"g": "h"},
	}
}

func changed(e *Entry) []string {
	var out []string

	for _, f := range e.Changed() {
		out = append(out, f.Short)
	}

	return out
}

func TestChanged(t *testing.T) {
	cases := []struct {
		name   string
		change func(*row)
		want   string
	}{
		{"nothing", func(*row) {}, ""},
		{"plain field", func(r *row) { r.Name = "x" }, "Name"},
		{"set to zero", func(r *row) { r.Name = "" }, "Name"},
		{"pointer replaced", func(r *row) { r.Bio = new("x") }, "Bio"},
		{"pointer replaced with equal value", func(r *row) { r.Bio = new("b") }, ""},
		{"pointer set to nil", func(r *row) { r.Bio = nil }, "Bio"},
		{"written through pointer", func(r *row) { *r.Bio = "x" }, "Bio"},
		{"bytes edited in place", func(r *row) { r.Data[0] = 'x' }, "Data"},
		{"slice edited in place", func(r *row) { r.Tags[0] = "x" }, "Tags"},
		{"slice appended", func(r *row) { r.Tags = append(r.Tags, "x") }, "Tags"},
		{"map edited in place", func(r *row) { r.Attrs["g"] = "x" }, "Attrs"},
		{"nested struct field", func(r *row) { r.Home.City = "x" }, "Home"},
		{"slice inside struct", func(r *row) { r.Home.Tags[0] = "x" }, "Home"},
		{
			"same instant in another zone",
			func(r *row) { r.At = r.At.In(time.FixedZone("x", 3600)) },
			"",
		},
		{"another instant", func(r *row) { r.At = r.At.Add(time.Second) }, "At"},
		{"the key is not a change", func(r *row) { r.ID = 2 }, ""},
		{"a relation is not a change", func(r *row) { r.Peers = []row{{}} }, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := fresh()
			e := New().Track(meta.Of[row](), unsafe.Pointer(r), Unchanged)
			c.change(r)

			got := ""

			if names := changed(e); len(names) > 0 {
				got = names[0]
			}

			if got != c.want {
				t.Errorf("changed = %q, want %q", got, c.want)
			}
		})
	}
}

func TestSavedAndUndo(t *testing.T) {
	r := fresh()
	e := New().Track(meta.Of[row](), unsafe.Pointer(r), Unchanged)
	r.Name = "x"

	undo := e.Saved(Unchanged)

	if len(e.Changed()) != 0 {
		t.Error("a saved value still counts as changed")
	}

	undo()

	if got := changed(e); len(got) != 1 || got[0] != "Name" {
		t.Errorf("after undo: %v", got)
	}
}

func TestLoadReturnsTheFirstInstance(t *testing.T) {
	tr, m := New(), meta.Of[row]()

	a, b := fresh(), fresh()
	if got := tr.Load(m, unsafe.Pointer(a)); got != unsafe.Pointer(a) {
		t.Fatal("first load replaced")
	}

	if got := tr.Load(m, unsafe.Pointer(b)); got != unsafe.Pointer(a) {
		t.Fatal("second load not resolved to the first")
	}

	if tr.Len() != 1 {
		t.Errorf("len = %d", tr.Len())
	}
}

func TestCopyKeepsBothInstances(t *testing.T) {
	tr, m := New(), meta.Of[row]()
	a, b := fresh(), fresh()
	tr.Load(m, unsafe.Pointer(a))
	tr.Copy(m, unsafe.Pointer(b))

	if tr.Len() != 2 {
		t.Fatalf("len = %d", tr.Len())
	}

	if got := tr.Find(m, int64(1)); got == nil || got.Entity != unsafe.Pointer(a) {
		t.Error("identity moved to the copy")
	}
}

func TestRemoveMovesNothingElse(t *testing.T) {
	tr, m := New(), meta.Of[row]()
	a := fresh()
	tr.Track(m, unsafe.Pointer(a), Unchanged)
	tr.Remove(unsafe.Pointer(a))

	if tr.Len() != 0 || tr.Find(m, int64(1)) != nil {
		t.Error("entry survived Remove")
	}

	tr.Remove(unsafe.Pointer(a)) // a second Remove is harmless
}

func TestNoKeyIsNotTracked(t *testing.T) {
	type keyless struct{ Name string }

	tr, m := New(), meta.Of[keyless]()

	v := &keyless{"a"}
	if got := tr.Load(m, unsafe.Pointer(v)); got != unsafe.Pointer(v) || tr.Len() != 0 {
		t.Error("keyless entity was tracked")
	}
}

func TestEntriesAreOldestFirst(t *testing.T) {
	tr, m := New(), meta.Of[row]()

	for i := range int64(50) {
		r := fresh()
		r.ID = 100 - i
		tr.Track(m, unsafe.Pointer(r), Unchanged)
	}

	entries := tr.Entries()
	for i := 1; i < len(entries); i++ {
		if entries[i-1].seq >= entries[i].seq {
			t.Fatal("out of order")
		}
	}
}

func TestMoveFollowsTheEntity(t *testing.T) {
	tr, m := New(), meta.Of[row]()
	a := fresh()
	e := tr.Track(m, unsafe.Pointer(a), Unchanged)
	b := *a
	b.Name = "changed in the new location"
	tr.Move(e, unsafe.Pointer(&b))

	if tr.Get(unsafe.Pointer(a)) != nil || tr.Get(unsafe.Pointer(&b)) != e {
		t.Error("the address map did not move")
	}

	if got := changed(e); len(got) != 1 || got[0] != "Name" {
		t.Errorf("changed = %v", got)
	}
}

func TestForgetUsesTheStoredKey(t *testing.T) {
	tr, m := New(), meta.Of[row]()
	a := fresh()
	e := tr.Track(m, unsafe.Pointer(a), Unchanged)
	a.ID = 99 // memory now describes another row

	tr.Forget(e)

	if tr.Find(m, int64(1)) != nil || tr.Len() != 0 {
		t.Error("entry survived Forget")
	}
}

func TestScratchIsPrivate(t *testing.T) {
	tr, m := New(), meta.Of[row]()
	a := fresh()
	e := tr.Track(m, unsafe.Pointer(a), Unchanged)
	scratch := (*row)(e.Scratch())
	*scratch.Bio = "x"
	scratch.Name = "x"

	if len(e.Changed()) != 0 {
		t.Error("changing the scratch copy changed the snapshot")
	}
}

func TestKeyChanged(t *testing.T) {
	a := fresh()

	e := New().Track(meta.Of[row](), unsafe.Pointer(a), Unchanged)
	if e.KeyChanged() {
		t.Error("key reported changed")
	}

	a.ID = 2

	if !e.KeyChanged() {
		t.Error("key change missed")
	}
}
