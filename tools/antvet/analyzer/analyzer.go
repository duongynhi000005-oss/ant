// Package analyzer finds mistakes in ant code that compile but fail at run time.
// A probe is a value from ant.Fields, and everything here follows from that:
// the compiler checks a field's value type, but only a look at the source can tell
// that a pointer came from a probe, or from the right model.
package analyzer

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// antPath is the import path of ant.
const antPath = "github.com/skudra-go/ant"

// Analyzer reports field pointers that are not probes, probes of the wrong model,
// and Find keys of the wrong type.
var Analyzer = &analysis.Analyzer{
	Name:      "antvet",
	Doc:       "check ant queries for what the compiler cannot see",
	Requires:  []*analysis.Analyzer{inspect.Analyzer},
	Run:       run,
	FactTypes: []analysis.Fact{new(probeFact)},
}

// probeFact marks a package variable as a probe, across packages.
type probeFact struct {
	Model string
}

// AFact implements analysis.Fact.
func (*probeFact) AFact() {}

func (f *probeFact) String() string { return "probe of " + f.Model }

// fieldTaking lists the functions whose first argument is a field pointer.
var fieldTaking = map[string]bool{
	"Eq":        true,
	"Ne":        true,
	"Gt":        true,
	"Gte":       true,
	"Lt":        true,
	"Lte":       true,
	"In":        true,
	"NotIn":     true,
	"Between":   true,
	"Like":      true,
	"ILike":     true,
	"IsNull":    true,
	"IsNotNull": true,
	"Set":       true,
	"SetExpr":   true,
	"Inc":       true,
	"Dec":       true,
	"Col":       true,
	"F":         true,
}

func run(pass *analysis.Pass) (any, error) {
	findProbes(pass)
	in := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	locals := map[types.Object]string{}

	in.Preorder([]ast.Node{(*ast.AssignStmt)(nil)}, func(n ast.Node) {
		assign := n.(*ast.AssignStmt)
		if len(assign.Lhs) != len(assign.Rhs) {
			return
		}

		for i, left := range assign.Lhs {
			id, ok := left.(*ast.Ident)
			if !ok {
				continue
			}

			if model := probeSource(pass, assign.Rhs[i]); model != "" {
				if object := pass.TypesInfo.Defs[id]; object != nil {
					locals[object] = model
				}
			}
		}
	})

	c := &checker{pass, locals}

	in.WithStack(
		[]ast.Node{(*ast.CallExpr)(nil)},
		func(n ast.Node, push bool, stack []ast.Node) bool {
			if !push {
				return true
			}

			call := n.(*ast.CallExpr)
			c.checkCondition(call)
			c.checkWhere(call, stack)
			checkFind(pass, call)

			return true
		},
	)

	return nil, nil
}

// checker holds what the checks share.
type checker struct {
	pass   *analysis.Pass
	locals map[types.Object]string // probes in local variables
}

// findProbes records package variables set from ant.Fields, or from another probe.
func findProbes(pass *analysis.Pass) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok || len(value.Names) != len(value.Values) {
					continue
				}

				for i, name := range value.Names {
					object := pass.TypesInfo.Defs[name]
					if object == nil {
						continue
					}

					if model := probeSource(pass, value.Values[i]); model != "" {
						pass.ExportObjectFact(object, &probeFact{model})
					}
				}
			}
		}
	}
}

// probeSource returns the model of an expression that is a probe, or "".
func probeSource(pass *analysis.Pass, e ast.Expr) string {
	switch e := ast.Unparen(e).(type) {
	case *ast.CallExpr:
		if isAntCall(pass, e, "Fields") {
			if model := modelOf(pass.TypesInfo.TypeOf(e)); model != "" {
				return model
			}
		}
	case *ast.Ident, *ast.SelectorExpr:
		if object := objectOf(pass, e); object != nil {
			fact := new(probeFact)
			if pass.ImportObjectFact(object, fact) {
				return fact.Model
			}
		}
	}

	return ""
}

func modelOf(t types.Type) string {
	pointer, ok := t.(*types.Pointer)
	if !ok {
		return ""
	}

	return types.TypeString(pointer.Elem(), nil)
}

func objectOf(pass *analysis.Pass, e ast.Expr) types.Object {
	switch e := e.(type) {
	case *ast.Ident:
		return pass.TypesInfo.Uses[e]
	case *ast.SelectorExpr:
		return pass.TypesInfo.Uses[e.Sel]
	}

	return nil
}

func isAntCall(pass *analysis.Pass, call *ast.CallExpr, name string) bool {
	return calleeName(pass, call) == name
}

// calleeName returns the name of an ant package function being called, or "".
func calleeName(pass *analysis.Pass, call *ast.CallExpr) string {
	fun := ast.Unparen(call.Fun)
	switch index := fun.(type) {
	case *ast.IndexExpr:
		fun = index.X
	case *ast.IndexListExpr:
		fun = index.X
	}

	selector, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	object, ok := pass.TypesInfo.Uses[selector.Sel].(*types.Func)
	if !ok || object.Pkg() == nil || object.Pkg().Path() != antPath {
		return ""
	}

	if object.Type().(*types.Signature).Recv() != nil {
		return ""
	}

	return object.Name()
}

// probeOf follows a pointer like &U.Age to its root, and returns the root's
// variable and whether that is a probe.
func (c *checker) probeOf(arg ast.Expr) (root ast.Expr, model string, isField bool) {
	pass := c.pass

	unary, ok := ast.Unparen(arg).(*ast.UnaryExpr)
	if !ok || unary.Op.String() != "&" {
		return nil, "", false
	}

	e := ast.Unparen(unary.X)

	for {
		selector, ok := e.(*ast.SelectorExpr)
		if !ok {
			break
		}

		object, isPackage := pass.TypesInfo.Uses[identOf(selector.X)].(*types.PkgName)
		if isPackage && object != nil {
			break
		}

		e = ast.Unparen(selector.X)
	}

	object := objectOf(pass, e)
	if object == nil {
		return e, "", true
	}

	if model := c.locals[object]; model != "" {
		return e, model, true
	}

	fact := new(probeFact)
	if pass.ImportObjectFact(object, fact) {
		return e, fact.Model, true
	}

	return e, "", true
}

func identOf(e ast.Expr) *ast.Ident {
	id, _ := e.(*ast.Ident)

	return id
}

// checkCondition reports a field pointer that is not from a probe.
func (c *checker) checkCondition(call *ast.CallExpr) {
	pass := c.pass

	name := calleeName(pass, call)
	if !fieldTaking[name] || len(call.Args) == 0 {
		return
	}
	// A pointer that is not written &x.y here came from elsewhere, and may be fine.
	root, model, isField := c.probeOf(call.Args[0])
	if !isField && root == nil {
		if unary, ok := ast.Unparen(call.Args[0]).(*ast.UnaryExpr); ok && unary.Op.String() == "&" {
			pass.Reportf(
				call.Args[0].Pos(),
				"ant.%s needs a field of a probe from ant.Fields",
				name,
			)
		}

		return
	}

	if model == "" {
		pass.Reportf(
			call.Args[0].Pos(),
			"ant.%s needs a field of a probe from ant.Fields, not of an ordinary variable",
			name,
		)
	}
}

// queryModel returns T of a *Query[T] or *Table[T] receiver, or "".
func queryModel(t types.Type) string {
	if pointer, ok := t.(*types.Pointer); ok {
		t = pointer.Elem()
	}

	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != antPath {
		return ""
	}

	if name := named.Obj().
		Name(); (name != "Query" && name != "Table") ||
		named.TypeArgs().Len() != 1 {
		return ""
	}

	return types.TypeString(named.TypeArgs().At(0), nil)
}

// singleModel lists conditions that name one column of the query's own model.
// Those that compare two columns can refer to an outer query.
var singleModel = map[string]bool{
	"Eq":        true,
	"Ne":        true,
	"Gt":        true,
	"Gte":       true,
	"Lt":        true,
	"Lte":       true,
	"In":        true,
	"NotIn":     true,
	"Between":   true,
	"Like":      true,
	"ILike":     true,
	"IsNull":    true,
	"IsNotNull": true,
}

// checkWhere reports a condition on a model other than the query's.
func (c *checker) checkWhere(call *ast.CallExpr, stack []ast.Node) {
	pass := c.pass

	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Where" {
		return
	}

	model := queryModel(pass.TypesInfo.TypeOf(selector.X))
	if model == "" || joinsLater(stack) {
		return
	}

	for _, arg := range call.Args {
		condition, ok := ast.Unparen(arg).(*ast.CallExpr)
		if !ok || !singleModel[calleeName(pass, condition)] || len(condition.Args) == 0 {
			continue
		}

		if _, probe, _ := c.probeOf(condition.Args[0]); probe != "" && probe != model {
			pass.Reportf(
				condition.Pos(),
				"condition on %s in a query of %s",
				short(probe),
				short(model),
			)
		}
	}
}

// joinsLater reports whether the chain goes on to a join, since a condition can
// come before the join that brings its model in.
func joinsLater(stack []ast.Node) bool {
	for _, n := range stack {
		if selector, ok := n.(*ast.SelectorExpr); ok &&
			strings.Contains(selector.Sel.Name, "Join") {
			return true
		}
	}

	return false
}

func short(name string) string { return name[strings.LastIndex(name, ".")+1:] }

// checkFind reports a Find key of the wrong type, for a model whose key is its ID
// field.
func checkFind(pass *analysis.Pass, call *ast.CallExpr) {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Find" || len(call.Args) != 2 {
		return
	}

	receiver := pass.TypesInfo.TypeOf(selector.X)
	if queryModel(receiver) == "" {
		return
	}

	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = pointer.Elem()
	}

	model := receiver.(*types.Named).TypeArgs().At(0)
	object, _, _ := types.LookupFieldOrMethod(model, true, pass.Pkg, "ID")

	id, ok := object.(*types.Var)
	if !ok {
		return
	}

	key := pass.TypesInfo.TypeOf(call.Args[1])
	if key == nil || compatible(id.Type(), key) {
		return
	}

	pass.Reportf(
		call.Args[1].Pos(),
		"Find key is %s, and %s has a key of type %s",
		types.TypeString(key, nil),
		short(types.TypeString(model, nil)),
		types.TypeString(id.Type(), nil),
	)
}

func compatible(want, got types.Type) bool {
	if types.Identical(want, got) {
		return true
	}

	a, aok := want.Underlying().(*types.Basic)

	b, bok := got.Underlying().(*types.Basic)
	if !aok || !bok {
		return false
	}

	both := func(info types.BasicInfo) bool { return a.Info()&info != 0 && b.Info()&info != 0 }

	return both(types.IsInteger) || both(types.IsString)
}
