package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"slices"
	"sort"
	"unicode/utf8"
)

// minFields is the keyed fields a literal can hold before each needs its own line.
const minFields = 4

// maxWidth is the longest line a one-line function can be on, in the columns that
// golines counts, with a tab as four. It matches max-len in .golangci.yml.
const maxWidth = 100

// passes bounds the rewriting. Spreading a literal can make the one around it
// span lines, so a few rounds are normal, and this many means a bug.
const passes = 16

// Format returns src with wrapped lists spread out, one item per line. It only
// adds line breaks and trailing commas, so the code means what it meant.
func Format(src []byte) ([]byte, error) {
	for range passes {
		formatted, err := format.Source(src)
		if err != nil {
			return nil, err
		}

		edits, err := collect(formatted)
		if err != nil {
			return nil, err
		}

		if len(edits) == 0 {
			return formatted, nil
		}

		src = apply(formatted, edits)
	}

	return nil, errors.New("antfmt: the code did not settle")
}

// edit inserts text before the byte at offset.
type edit struct {
	offset int
	text   string
}

func apply(src []byte, edits []edit) []byte {
	sort.SliceStable(edits, func(i, j int) bool {
		return edits[i].offset < edits[j].offset
	})

	out := make([]byte, 0, len(src)+len(edits)*2)
	last := 0

	for _, e := range edits {
		out = append(out, src[last:e.offset]...)
		out = append(out, e.text...)
		last = e.offset
	}

	return append(out, src[last:]...)
}

type collector struct {
	file  *token.File
	lines [][]byte
	edits []edit
}

func collect(src []byte) ([]edit, error) {
	files := token.NewFileSet()

	parsed, err := parser.ParseFile(
		files,
		"",
		src,
		parser.ParseComments|parser.SkipObjectResolution,
	)
	if err != nil {
		return nil, err
	}

	c := &collector{file: files.File(parsed.Pos()), lines: bytes.Split(src, []byte("\n"))}

	if ast.IsGenerated(parsed) {
		return nil, nil
	}

	ast.Inspect(parsed, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CompositeLit:
			c.literal(n)
		case *ast.CallExpr:
			c.wrapped(n.Lparen, n.Args, n.Rparen)
		case *ast.FuncType:
			c.fields(n.Params)
			c.fields(n.Results)
		case *ast.FuncDecl:
			c.oneLiner(n.Body)
		case *ast.FuncLit:
			c.oneLiner(n.Body)
		case *ast.BlockStmt:
			c.separate(n.List)
		case *ast.CaseClause:
			c.separate(n.Body)
		case *ast.CommClause:
			c.separate(n.Body)
		}

		return true
	})

	return c.edits, nil
}

func (c *collector) line(p token.Pos) int { return c.file.Line(p) }

// literal spreads a literal with many keyed fields, one with keyed fields that
// already spans lines, and any other that wraps part of the way.
func (c *collector) literal(n *ast.CompositeLit) {
	if !keyed(n.Elts) {
		c.wrapped(n.Lbrace, n.Elts, n.Rbrace)

		return
	}

	if len(n.Elts) >= minFields || c.line(n.Lbrace) != c.line(n.Rbrace) {
		c.spread(n.Lbrace, n.Elts, n.Rbrace)
	}
}

func keyed(items []ast.Expr) bool {
	if len(items) == 0 {
		return false
	}

	return !slices.ContainsFunc(items, func(e ast.Expr) bool {
		_, ok := e.(*ast.KeyValueExpr)

		return !ok
	})
}

func (c *collector) fields(list *ast.FieldList) {
	if list == nil || !list.Opening.IsValid() || len(list.List) == 0 {
		return
	}

	c.wrapped(list.Opening, list.List, list.Closing)
}

// wrapped spreads a list that breaks across lines with several items left on one
// of them. A list that starts on one line stays, which leaves the common shape of
// a function literal or a struct as the last argument alone.
func (c *collector) wrapped(open token.Pos, items any, end token.Pos) {
	nodes := nodesOf(items)
	if len(nodes) < 2 || c.line(nodes[0].Pos()) == c.line(nodes[len(nodes)-1].Pos()) {
		return
	}

	for i := 1; i < len(nodes); i++ {
		if c.line(nodes[i-1].End()) == c.line(nodes[i].Pos()) {
			c.spread(open, items, end)

			return
		}
	}
}

// spread puts each item on its own line, and the closing token on the next.
func (c *collector) spread(open token.Pos, items any, end token.Pos) {
	nodes := nodesOf(items)
	line := c.line(open)

	for _, node := range nodes {
		if c.line(node.Pos()) == line {
			c.insert(node.Pos(), "\n")
		}

		line = c.line(node.End())
	}

	if c.line(end) == line {
		c.insert(end, ",\n")
	}
}

func (c *collector) insert(p token.Pos, text string) {
	c.edits = append(c.edits, edit{offset: c.file.Offset(p), text: text})
}

// oneLiner opens up a function written on one line, if that line is too long.
func (c *collector) oneLiner(body *ast.BlockStmt) {
	if body == nil || len(body.List) != 1 {
		return
	}

	line := c.line(body.Lbrace)
	if line == c.line(body.Rbrace) && c.width(line) > maxWidth {
		c.insert(body.List[0].Pos(), "\n")
		c.insert(body.Rbrace, "\n")
	}
}

func (c *collector) width(line int) int {
	text := c.lines[line-1]

	return utf8.RuneCount(text) + 3*bytes.Count(text, []byte("\t"))
}

// separate puts a blank line after a plain statement unless what follows belongs
// to it. wsl covers much of this, but lets an if through under a statement it has
// nothing to do with.
func (c *collector) separate(list []ast.Stmt) {
	for i := 1; i < len(list); i++ {
		a, b := list[i-1], list[i]

		if plain(a) && c.line(b.Pos())-c.line(a.End()) == 1 && !c.together(a, b) {
			c.insert(b.Pos(), "\n")
		}
	}
}

// together reports whether b can sit directly under a. An if, for, or switch can
// if its header reads what a made, and after a statement of several lines only if
// that is a check of err or ok. Plain statements can if they are one line each, or
// if b reads what a made, like require.NoError(t, err) under the call.
func (c *collector) together(a, b ast.Stmt) bool {
	names := defined(a)
	spans := c.line(a.Pos()) != c.line(a.End())

	if header, ok := headerOf(b); ok {
		if spans {
			names = slices.DeleteFunc(names, func(name string) bool {
				return name != "err" && name != "ok"
			})
		}

		return reads(header, names)
	}

	if !plain(b) {
		return true
	}

	single := !spans && c.line(b.Pos()) == c.line(b.End())

	return single || reads([]ast.Node{b}, names)
}

// headerOf returns what a block statement evaluates before its body.
func headerOf(s ast.Stmt) ([]ast.Node, bool) {
	var out []ast.Node

	switch s := s.(type) {
	case *ast.IfStmt:
		if s.Init != nil {
			out = append(out, s.Init)
		}

		out = append(out, s.Cond)
	case *ast.ForStmt:
		if s.Init != nil {
			out = append(out, s.Init)
		}

		if s.Cond != nil {
			out = append(out, s.Cond)
		}
	case *ast.RangeStmt:
		out = append(out, s.X)
	case *ast.SwitchStmt:
		if s.Init != nil {
			out = append(out, s.Init)
		}

		if s.Tag != nil {
			out = append(out, s.Tag)
		}
	case *ast.TypeSwitchStmt:
		out = append(out, s.Assign)
	default:
		return nil, false
	}

	return out, true
}

func plain(s ast.Stmt) bool {
	switch s.(type) {
	case *ast.AssignStmt, *ast.DeclStmt, *ast.ExprStmt, *ast.IncDecStmt, *ast.SendStmt:
		return true
	}

	return false
}

// defined returns the names a statement assigns.
func defined(s ast.Stmt) []string {
	var out []string

	switch s := s.(type) {
	case *ast.AssignStmt:
		for _, e := range s.Lhs {
			if id, ok := e.(*ast.Ident); ok && id.Name != "_" {
				out = append(out, id.Name)
			}
		}
	case *ast.IncDecStmt:
		if id, ok := s.X.(*ast.Ident); ok {
			out = append(out, id.Name)
		}
	case *ast.DeclStmt:
		if gen, ok := s.Decl.(*ast.GenDecl); ok {
			for _, spec := range gen.Specs {
				if value, ok := spec.(*ast.ValueSpec); ok {
					for _, id := range value.Names {
						out = append(out, id.Name)
					}
				}
			}
		}
	}

	return out
}

func reads(nodes []ast.Node, names []string) bool {
	found := false

	for _, node := range nodes {
		ast.Inspect(node, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && slices.Contains(names, id.Name) {
				found = true
			}

			return !found
		})
	}

	return found
}

func nodesOf(items any) []ast.Node {
	var out []ast.Node

	switch items := items.(type) {
	case []ast.Expr:
		for _, item := range items {
			out = append(out, item)
		}
	case []*ast.Field:
		for _, item := range items {
			out = append(out, item)
		}
	}

	return out
}
