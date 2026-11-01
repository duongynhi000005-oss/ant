// Package api keeps a record of the public API, so a change to it is a decision
// someone reviews, and not an accident. To accept a change:
//
//	go test ./tests/api -update
package api

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite docs/api.txt")

// packages lists the public packages, from the module root.
var packages = []string{
	".",
	"anttest",
	"dialect",
	"dialect/postgres",
	"dialect/postgres/pgxext",
	"ids",
	"metrics",
	"migrate",
	"qb",
	"schema",
	"secure",
	"validate",
}

var unwrap = strings.NewReplacer("( ", "(", ", )", ")")

func TestPublicAPI(t *testing.T) {
	root := filepath.Join("..", "..")

	var lines []string

	for _, pkg := range packages {
		lines = append(lines, declarations(t, root, pkg)...)
	}

	got := strings.Join(lines, "\n") + "\n"
	file := filepath.Join(root, "docs", "api.txt")

	if *update {
		require.NoError(t, os.WriteFile(file, []byte(got), 0o644))

		return
	}

	want, err := os.ReadFile(file)
	require.NoError(t, err)

	require.Equal(
		t,
		string(want),
		got,
		"the public API changed: review it, then run go test ./tests/api -update",
	)
}

// declarations lists a package's exported signatures, sorted.
func declarations(t *testing.T, root, pkg string) []string {
	t.Helper()

	fset := token.NewFileSet()
	entries, err := os.ReadDir(filepath.Join(root, pkg))
	require.NoError(t, err)

	var out []string

	render := func(node any) string {
		var b strings.Builder
		require.NoError(t, printer.Fprint(&b, fset, node))
		// A long signature wraps, which is layout and not a change to the API.
		return unwrap.Replace(strings.Join(strings.Fields(b.String()), " "))
	}

	name := pkg
	if name == "." {
		name = "ant"
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(root, pkg, entry.Name()), nil, 0)
		require.NoError(t, err)

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() || (d.Recv != nil && !exportedReceiver(d.Recv)) {
					continue
				}

				d.Body = nil
				out = append(out, name+": "+render(d))
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							out = append(out, fmt.Sprintf("%s: type %s", name, render(s.Name)))
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.IsExported() {
								out = append(out, fmt.Sprintf("%s: %s %s", name, d.Tok, n.Name))
							}
						}
					}
				}
			}
		}
	}

	slices.Sort(out)

	return out
}

func exportedReceiver(list *ast.FieldList) bool {
	e := list.List[0].Type

	for {
		switch x := e.(type) {
		case *ast.StarExpr:
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.IndexListExpr:
			e = x.X
		case *ast.Ident:
			return x.IsExported()
		default:
			return false
		}
	}
}
