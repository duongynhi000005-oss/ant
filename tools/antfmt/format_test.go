package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	for _, tt := range []struct {
		name, in, want string
	}{
		{
			name: "four keyed fields on one line",
			in:   `var x = T{A: 1, B: 2, C: 3, D: 4}`,
			want: "var x = T{\n\tA: 1,\n\tB: 2,\n\tC: 3,\n\tD: 4,\n}",
		},
		{
			name: "three keyed fields stay",
			in:   `var x = T{A: 1, B: 2, C: 3}`,
			want: `var x = T{A: 1, B: 2, C: 3}`,
		},
		{
			name: "keyed fields wrapped in groups",
			in:   "var x = T{\n\tA: 1, B: 2,\n\tC: 3,\n}",
			want: "var x = T{\n\tA: 1,\n\tB: 2,\n\tC: 3,\n}",
		},
		{
			name: "a field holding a long literal",
			in:   "var x = T{A: 1, B: U{\n\tC: 2,\n}}",
			want: "var x = T{\n\tA: 1,\n\tB: U{\n\t\tC: 2,\n\t},\n}",
		},
		{
			name: "a nested literal",
			in:   `var x = f(T{A: 1, B: 2, C: 3, D: 4})`,
			want: "var x = f(T{\n\tA: 1,\n\tB: 2,\n\tC: 3,\n\tD: 4,\n})",
		},
		{
			name: "positional rows stay",
			in:   "var x = []T{\n\t{1, 2, 3, 4, 5},\n\t{6, 7, 8, 9, 0},\n}",
			want: "var x = []T{\n\t{1, 2, 3, 4, 5},\n\t{6, 7, 8, 9, 0},\n}",
		},
		{
			name: "a positional list wrapped in groups",
			in:   "var x = []T{a, b,\n\tc, d}",
			want: "var x = []T{\n\ta,\n\tb,\n\tc,\n\td,\n}",
		},
		{
			name: "wrapped arguments",
			in:   "var x = f(a, b,\n\tc)",
			want: "var x = f(\n\ta,\n\tb,\n\tc,\n)",
		},
		{
			name: "a function literal last stays",
			in:   "var x = f(a, b, func() {\n\tg()\n})",
			want: "var x = f(a, b, func() {\n\tg()\n})",
		},
		{
			name: "variadic arguments",
			in:   "var x = f(a, b,\n\tc...)",
			want: "var x = f(\n\ta,\n\tb,\n\tc...,\n)",
		},
		{
			name: "wrapped parameters",
			in:   "func f(a int, b int,\n\tc int) {\n}",
			want: "func f(\n\ta int,\n\tb int,\n\tc int,\n) {\n}",
		},
		{
			name: "multi-line statements get room",
			in:   "func f() {\n\ta := g(\n\t\t1,\n\t)\n\tb := 2\n\tc := 3\n\th(\n\t\t4,\n\t)\n}",
			want: "func f() {\n\ta := g(\n\t\t1,\n\t)\n\n\tb := 2\n\tc := 3\n\n\th(\n\t\t4,\n\t)\n}",
		},
		{
			name: "a check of what the statement made stays close",
			in:   "func f() {\n\ta, err := g(\n\t\t1,\n\t)\n\tcheck(t, err)\n\tuse(a)\n}",
			want: "func f() {\n\ta, err := g(\n\t\t1,\n\t)\n\tcheck(t, err)\n\tuse(a)\n}",
		},
		{
			name: "one-line statements stay together",
			in:   "func f() {\n\ta := 1\n\tb := 2\n\tg(a, b)\n}",
			want: "func f() {\n\ta := 1\n\tb := 2\n\tg(a, b)\n}",
		},
		{
			name: "a comment keeps its statement",
			in:   "func f() {\n\ta := g(\n\t\t1,\n\t)\n\t// about b\n\tb := 2\n}",
			want: "func f() {\n\ta := g(\n\t\t1,\n\t)\n\t// about b\n\tb := 2\n}",
		},
		{
			name: "a long one-line function opens up",
			in:   "var x = f(func(value string) { g(value, \"a string long enough to go over the limit on a line here\") })",
			want: "var x = f(func(value string) {\n\tg(value, \"a string long enough to go over the limit on a line here\")\n})",
		},
		{
			name: "a short one-line function stays",
			in:   "func f() int { return 1 }",
			want: "func f() int { return 1 }",
		},
		{
			name: "an if under something unrelated",
			in:   "func f() {\n\ta := 1\n\tif b {\n\t}\n}",
			want: "func f() {\n\ta := 1\n\n\tif b {\n\t}\n}",
		},
		{
			name: "an if that checks what was just made",
			in:   "func f() {\n\ta := g()\n\tif a == nil {\n\t}\n}",
			want: "func f() {\n\ta := g()\n\tif a == nil {\n\t}\n}",
		},
		{
			name: "an if used only inside its body",
			in:   "func f() {\n\ta := 1\n\tfor _, x := range xs {\n\t\ta += x\n\t}\n}",
			want: "func f() {\n\ta := 1\n\n\tfor _, x := range xs {\n\t\ta += x\n\t}\n}",
		},
		{
			name: "an if under a multi-line statement",
			in:   "func f() {\n\ta := T{\n\t\tB: 1,\n\t}\n\tif a.B == 1 {\n\t}\n}",
			want: "func f() {\n\ta := T{\n\t\tB: 1,\n\t}\n\n\tif a.B == 1 {\n\t}\n}",
		},
		{
			name: "an error check under a multi-line statement",
			in:   "func f() {\n\ta, err := g(\n\t\t1,\n\t)\n\tif err != nil {\n\t}\n}",
			want: "func f() {\n\ta, err := g(\n\t\t1,\n\t)\n\tif err != nil {\n\t}\n}",
		},
		{
			name: "a switch on what was just made",
			in:   "func f() {\n\ta := g()\n\tswitch a {\n\t}\n}",
			want: "func f() {\n\ta := g()\n\tswitch a {\n\t}\n}",
		},
		{
			name: "a range under something unrelated",
			in:   "func f() {\n\tg()\n\tfor range xs {\n\t}\n}",
			want: "func f() {\n\tg()\n\n\tfor range xs {\n\t}\n}",
		},
		{
			name: "comments survive",
			in:   "var x = T{\n\tA: 1, B: 2, // two\n\tC: 3, // three\n}",
			want: "var x = T{\n\tA: 1,\n\tB: 2, // two\n\tC: 3, // three\n}",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Format([]byte("package p\n\n" + tt.in + "\n"))
			if err != nil {
				t.Fatal(err)
			}

			if want := "package p\n\n" + tt.want + "\n"; string(got) != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}

			again, err := Format(got)
			if err != nil {
				t.Fatal(err)
			}

			if string(again) != string(got) {
				t.Errorf("a second run changed it:\n%s", again)
			}
		})
	}
}

func TestGeneratedFilesStay(t *testing.T) {
	src := "// Code generated by x. DO NOT EDIT.\n\npackage p\n\nvar x = T{A: 1, B: 2, C: 3, D: 4}\n"

	got, err := Format([]byte(src))
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != src {
		t.Errorf("got:\n%s", got)
	}
}

// Only line breaks and commas change, so both versions parse to the same tree.
func TestMeaningIsKept(t *testing.T) {
	src := `package p

func f(a int, b string,
	c ...bool) (x int, err error) {
	v := T{A: a, B: map[string]int{"k": 1, "l": 2, "m": 3, "n": 4}, C: []int{1, 2,
		3}, D: func() int { return 1 }()}
	g(a, b,
		c...)

	return v.A, nil
}
`

	got, err := Format([]byte(src))
	if err != nil {
		t.Fatal(err)
	}

	if a, b := shape(t, src), shape(t, string(got)); a != b {
		t.Errorf("the tree changed:\n%s\n---\n%s", a, b)
	}
}

// shape lists a file's nodes by kind and name, which positions do not affect.
func shape(t *testing.T, src string) string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	var out strings.Builder

	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case nil:
			out.WriteString(")")
		case *ast.Ident:
			out.WriteString(" " + n.Name + "(")
		case *ast.BasicLit:
			out.WriteString(" " + n.Value + "(")
		default:
			out.WriteString(" " + reflect.TypeOf(n).String() + "(")
		}

		return true
	})

	return out.String()
}
