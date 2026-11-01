package schema

import (
	"fmt"
	"slices"
	"strings"
)

// Hint is a possible rename that a diff saw but cannot be sure of. A rename and a
// drop plus an add look identical, and only a person knows which was meant.
type Hint struct {
	Table Ref
	// Column is empty when the hint is about the table itself.
	Column string
	To     string
}

// Flag returns the value for a rename in Diff that takes up this hint.
func (h Hint) Flag() string {
	if h.Column == "" {
		return h.Table.String() + "=" + h.To
	}

	return fmt.Sprintf("%s.%s=%s", h.Table, h.Column, h.To)
}

// Change returns the rename this hint suggests.
func (h Hint) Change() Change {
	if h.Column == "" {
		return RenameTable{Table: h.Table, To: h.To}
	}

	return RenameColumn{Table: h.Table, From: h.Column, To: h.To}
}

// String says what was seen, for a person to judge.
func (h Hint) String() string {
	if h.Column == "" {
		return fmt.Sprintf("table %s was removed and %s was added", h.Table, h.To)
	}

	return fmt.Sprintf("column %s.%s was removed and %s was added", h.Table, h.Column, h.To)
}

// Plan is the ordered list of changes from one schema to another.
type Plan struct {
	Changes []Change
	Hints   []Hint
}

// Empty reports whether there is nothing to do.
func (p *Plan) Empty() bool { return len(p.Changes) == 0 }

// Warnings returns the warnings of every change.
func (p *Plan) Warnings() []string {
	var out []string

	for _, c := range p.Changes {
		if w := c.Warning(); w != "" {
			out = append(out, w)
		}
	}

	return out
}

// Online reports whether any change must run outside a transaction.
func (p *Plan) Online() bool {
	return slices.ContainsFunc(p.Changes, func(c Change) bool {
		o, ok := c.(Online)

		return ok && o.RunsOnline()
	})
}

// phases hold changes in the order they must run. Foreign keys go first and
// return last, so that nothing depends on what changes in between, and renames
// come before everything, so that later steps can use the new names.
type phases struct {
	schemas, types, renames, dropKeys, dropRest, dropTables, dropColumns []Change
	createTables, addColumns, alterColumns, addRest, addKeys, comments   []Change
	dropTypes                                                            []Change
}

func (p *phases) plan() []Change {
	return slices.Concat(
		p.schemas,
		p.types,
		p.renames,
		p.dropKeys,
		p.dropRest,
		p.dropTables,
		p.dropColumns,
		p.createTables,
		p.addColumns,
		p.alterColumns,
		p.addRest,
		p.addKeys,
		p.comments,
		p.dropTypes,
	)
}

// compareTypes works out what changes in enums and sequences. They come before
// tables, which use them, and go after, once nothing does.
func (p *phases) compareTypes(old, to *Schema) {
	for _, e := range to.Enums {
		existing := old.Enum(e.Name)
		if existing == nil {
			p.types = append(p.types, CreateEnum{Enum: e})

			continue
		}

		for _, value := range e.Values {
			if !slices.Contains(existing.Values, value) {
				p.types = append(p.types, AddEnumValue{Enum: e.Name, Value: value})
			}
		}

		for _, value := range existing.Values {
			if !slices.Contains(e.Values, value) {
				p.types = append(p.types, RemoveEnumValue{Enum: e.Name, Value: value})
			}
		}
	}

	for _, e := range old.Enums {
		if to.Enum(e.Name) == nil {
			p.dropTypes = append(p.dropTypes, DropEnum{Enum: e.Name})
		}
	}

	for _, q := range to.Sequences {
		if old.Sequence(q.Name) == nil {
			p.types = append(p.types, CreateSequence{Sequence: q})
		}
	}

	for _, q := range old.Sequences {
		if to.Sequence(q.Name) == nil {
			p.dropTypes = append(p.dropTypes, DropSequence{Sequence: q.Name})
		}
	}
}

// Diff returns the plan from from to to. Either can be nil, meaning empty.
//
// Each rename says that something in from has a new name in to, as
// table=new_name or table.column=new_name, and a table can be written
// schema.table. A rename the diff cannot apply is an error, because the caller
// meant it to prevent a drop.
func Diff(from, to *Schema, renames ...string) (*Plan, error) {
	old := from.Clone()

	if to == nil {
		to = &Schema{}
	}

	var p phases

	for _, text := range renames {
		change, err := old.rename(text, to)
		if err != nil {
			return nil, err
		}

		p.renames = append(p.renames, change)
	}

	for _, name := range to.Schemas {
		if !slices.Contains(old.Schemas, name) {
			p.schemas = append(p.schemas, CreateSchema{Name: name})
		}
	}

	p.compareTypes(old, to)

	for _, t := range to.Tables {
		o := old.Table(t.Ref())
		if o == nil {
			p.createTable(t)

			continue
		}

		p.alterTable(o, t)
	}

	for _, o := range old.Tables {
		if to.Table(o.Ref()) == nil {
			for _, key := range o.ForeignKeys {
				p.dropKeys = append(p.dropKeys, DropForeignKey{Table: o.Ref(), Key: key})
			}

			p.dropTables = append(p.dropTables, DropTable{Table: o.Ref()})
		}
	}

	return &Plan{Changes: p.plan(), Hints: hints(old, to)}, nil
}

// rename applies a rename to s, as the database would.
func (s *Schema) rename(text string, to *Schema) (Change, error) {
	path, name, ok := strings.Cut(text, "=")
	if !ok || name == "" {
		return nil, fmt.Errorf("ant: rename %q: want table=new_name or table.column=new_name", text)
	}

	parts := strings.Split(path, ".")
	if len(parts) == 1 || len(parts) == 2 && s.Table(Ref{"", parts[0]}) == nil {
		ref := Ref{Name: parts[len(parts)-1]}

		if len(parts) == 2 {
			ref.Schema = parts[0]
		}

		if t := s.Table(ref); t != nil {
			return s.renameTable(t, name), nil
		}

		return nil, fmt.Errorf("ant: rename %q: no table %s in the snapshot", text, ref)
	}

	ref := Ref{Name: parts[0]}

	column := parts[len(parts)-1]

	if len(parts) == 3 {
		ref = Ref{parts[0], parts[1]}
	}

	t := s.Table(ref)
	if t == nil {
		return nil, fmt.Errorf("ant: rename %q: no table %s in the snapshot", text, ref)
	}

	c := t.Column(column)
	if c == nil {
		return nil, fmt.Errorf("ant: rename %q: no column %s in table %s", text, column, ref)
	}

	if target := to.Table(ref); target == nil || target.Column(name) == nil {
		return nil, fmt.Errorf("ant: rename %q: models have no column %s.%s", text, ref, name)
	}

	s.renameColumn(t, c, name)

	return RenameColumn{Table: ref, From: column, To: name}, nil
}

func (s *Schema) renameTable(t *Table, name string) Change {
	change := RenameTable{Table: t.Ref(), To: name}

	for _, other := range s.Tables {
		for _, key := range other.ForeignKeys {
			if key.RefSchema == t.Schema && key.RefTable == t.Name {
				key.RefTable = name
			}
		}
	}

	t.Name = name

	return change
}

func (s *Schema) renameColumn(t *Table, c *Column, name string) {
	swap := func(columns []string) {
		for i := range columns {
			if columns[i] == c.Name {
				columns[i] = name
			}
		}
	}

	if t.PrimaryKey != nil {
		swap(t.PrimaryKey.Columns)
	}

	for _, key := range t.Uniques {
		swap(key.Columns)
	}

	for _, index := range t.Indexes {
		swap(index.Include)

		for i := range index.Parts {
			if index.Parts[i].Column == c.Name {
				index.Parts[i].Column = name
			}
		}
	}

	for _, key := range t.ForeignKeys {
		swap(key.Columns)
	}

	for _, other := range s.Tables {
		for _, key := range other.ForeignKeys {
			if key.RefSchema == t.Schema && key.RefTable == t.Name {
				swap(key.RefColumns)
			}
		}
	}

	c.Name = name
}

func (p *phases) createTable(t *Table) {
	ref := t.Ref()

	p.createTables = append(p.createTables, CreateTable{Table: t})

	if t.Comment != "" {
		p.comments = append(p.comments, SetTableComment{Table: ref, Comment: t.Comment})
	}

	for _, c := range t.Columns {
		if c.Comment != "" {
			p.comments = append(
				p.comments,
				SetColumnComment{Table: ref, Column: c.Name, Comment: c.Comment},
			)
		}
	}

	for _, index := range t.Indexes {
		p.addRest = append(p.addRest, CreateIndex{Table: ref, Index: index, New: true})
	}

	for _, key := range t.ForeignKeys {
		p.addKeys = append(p.addKeys, AddForeignKey{Table: ref, Key: key, New: true})
	}
}

func (p *phases) alterTable(o, t *Table) {
	ref := t.Ref()

	if o.Comment != t.Comment {
		p.comments = append(p.comments, SetTableComment{Table: ref, Comment: t.Comment})
	}

	for _, c := range t.Columns {
		existing := o.Column(c.Name)

		switch {
		case existing == nil:
			p.addColumns = append(p.addColumns, AddColumn{Table: ref, Column: c})
		case !existing.sameAs(c):
			p.alterColumns = append(p.alterColumns, AlterColumn{Table: ref, From: existing, To: c})
		}

		if (existing == nil && c.Comment != "") ||
			(existing != nil && existing.Comment != c.Comment) {
			p.comments = append(
				p.comments,
				SetColumnComment{Table: ref, Column: c.Name, Comment: c.Comment},
			)
		}
	}

	for _, c := range o.Columns {
		if t.Column(c.Name) == nil {
			p.dropColumns = append(p.dropColumns, DropColumn{Table: ref, Column: c})
		}
	}

	switch {
	case o.PrimaryKey != nil && t.PrimaryKey == nil:
		p.dropRest = append(p.dropRest, DropPrimaryKey{Table: ref, Key: o.PrimaryKey})
	case o.PrimaryKey == nil && t.PrimaryKey != nil:
		p.addRest = append(p.addRest, AddPrimaryKey{Table: ref, Key: t.PrimaryKey})
	case o.PrimaryKey != nil && !o.PrimaryKey.sameAs(t.PrimaryKey):
		p.dropRest = append(p.dropRest, DropPrimaryKey{Table: ref, Key: o.PrimaryKey})
		p.addRest = append(p.addRest, AddPrimaryKey{Table: ref, Key: t.PrimaryKey})
	case o.PrimaryKey != nil && o.PrimaryKey.Name != t.PrimaryKey.Name:
		p.renames = append(
			p.renames,
			RenameConstraint{Table: ref, From: o.PrimaryKey.Name, To: t.PrimaryKey.Name},
		)
	}

	objects(
		o.Uniques,
		t.Uniques,
		func(k *Key) string { return k.Name },
		(*Key).sameAs,
		func(k *Key) { p.dropRest = append(p.dropRest, DropConstraint{Table: ref, Name: k.Name}) },
		func(k *Key) { p.addRest = append(p.addRest, AddUnique{Table: ref, Key: k}) },
		func(from, to string) {
			p.renames = append(p.renames, RenameConstraint{Table: ref, From: from, To: to})
		},
	)

	objects(
		o.Checks,
		t.Checks,
		func(c *Check) string { return c.Name },
		(*Check).sameAs,
		func(c *Check) {
			p.dropRest = append(p.dropRest, DropConstraint{Table: ref, Name: c.Name})
		},
		func(c *Check) { p.addRest = append(p.addRest, AddCheck{Table: ref, Check: c}) },
		func(from, to string) {
			p.renames = append(p.renames, RenameConstraint{Table: ref, From: from, To: to})
		},
	)

	objects(
		o.Indexes,
		t.Indexes,
		func(i *Index) string { return i.Name },
		(*Index).sameAs,
		func(i *Index) { p.dropRest = append(p.dropRest, DropIndex{Table: ref, Index: i}) },
		func(i *Index) { p.addRest = append(p.addRest, CreateIndex{Table: ref, Index: i}) },
		func(from, to string) {
			p.renames = append(p.renames, RenameIndex{Table: ref, From: from, To: to})
		},
	)

	objects(
		o.ForeignKeys,
		t.ForeignKeys,
		func(k *ForeignKey) string { return k.Name },
		(*ForeignKey).sameAs,
		func(k *ForeignKey) { p.dropKeys = append(p.dropKeys, DropForeignKey{Table: ref, Key: k}) },
		func(k *ForeignKey) { p.addKeys = append(p.addKeys, AddForeignKey{Table: ref, Key: k}) },
		func(from, to string) {
			p.renames = append(p.renames, RenameConstraint{Table: ref, From: from, To: to})
		},
	)
}

// objects compares two lists of named objects. An object with the same definition
// under a new name is renamed, so that an index on a large table is not rebuilt
// because its name changed.
func objects[T any](
	old, next []T,
	name func(T) string,
	same func(a, b T) bool,
	drop, add func(T),
	rename func(from, to string),
) {
	find := func(list []T, wanted string) (T, bool) {
		for _, item := range list {
			if name(item) == wanted {
				return item, true
			}
		}

		var zero T

		return zero, false
	}

	taken := map[string]bool{}

	var added []T

	for _, item := range next {
		if existing, ok := find(old, name(item)); ok {
			taken[name(existing)] = true

			if !same(existing, item) {
				drop(existing)
				add(item)
			}

			continue
		}

		added = append(added, item)
	}

	for _, item := range added {
		matched := false

		for _, existing := range old {
			if !taken[name(existing)] && same(existing, item) {
				taken[name(existing)] = true
				rename(name(existing), name(item))

				matched = true

				break
			}
		}

		if !matched {
			add(item)
		}
	}

	for _, existing := range old {
		if !taken[name(existing)] {
			drop(existing)
		}
	}
}

// hints finds what looks like a rename that nobody declared.
func hints(old, to *Schema) []Hint {
	var (
		out              []Hint
		dropped, created []*Table
	)

	for _, t := range old.Tables {
		if to.Table(t.Ref()) == nil {
			dropped = append(dropped, t)
		}
	}

	for _, t := range to.Tables {
		o := old.Table(t.Ref())
		if o == nil {
			created = append(created, t)

			continue
		}

		for _, gone := range o.Columns {
			if t.Column(gone.Name) != nil {
				continue
			}

			for _, added := range t.Columns {
				if o.Column(added.Name) == nil && added.Type == gone.Type {
					out = append(out, Hint{Table: t.Ref(), Column: gone.Name, To: added.Name})
				}
			}
		}
	}

	for _, gone := range dropped {
		for _, added := range created {
			if sameColumns(gone, added) {
				out = append(out, Hint{Table: gone.Ref(), To: added.Name})
			}
		}
	}

	return out
}

func sameColumns(a, b *Table) bool {
	return len(a.Columns) == len(b.Columns) &&
		!slices.ContainsFunc(a.Columns, func(c *Column) bool {
			other := b.Column(c.Name)

			return other == nil || other.Type != c.Type
		})
}
