// Package analyzer finds mistakes in ant code that compile but fail at run time.
// A probe is a value from ant.Fields, and everything here follows from that:
// the compiler checks a field's value type, but only a look at the source can tell
// that a pointer came from a probe, or from the right model.
package analyzer

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// antPath is the import path of ant.
const antPath = "github.com/skudra-go/ant"

// Analyzer reports field pointers that are not probes, probes of the wrong model,
// and Find keys of the wrong type. A comment //antvet:ignore on the line of a
// report, or on the line above it, silences it, for code that is wrong on purpose.
var Analyzer = &analysis.Analyzer{
	Name:      "antvet",
	Doc:       "check ant queries for what the compiler cannot see",
	Requires:  []*analysis.Analyzer{inspect.Analyzer},
	Run:       run,
	FactTypes: []analysis.Fact{new(probeFact), new(keyFact)},
}

// keyFact records the primary key a model's Configure method sets, which
// replaces the convention of a field named ID. Field is empty for a key Find
// cannot take, such as a composite one, or none.
type keyFact struct {
	Field string
}

// AFact implements analysis.Fact.
func (*keyFact) AFact() {}

func (f *keyFact) String() string { return "key " + f.Field }

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
	findKeys(pass)
	pass = quieted(pass)
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

// quieted returns pass with its reports dropped on lines that //antvet:ignore
// covers: its own, and the one below it.
func quieted(pass *analysis.Pass) *analysis.Pass {
	ignored := map[string]bool{}

	for _, file := range pass.Files {
		for _, group := range file.Comments {
			for _, comment := range group.List {
				if !strings.HasPrefix(comment.Text, "//antvet:ignore") {
					continue
				}

				at := pass.Fset.Position(comment.Pos())
				ignored[fmt.Sprintf("%s:%d", at.Filename, at.Line)] = true
				ignored[fmt.Sprintf("%s:%d", at.Filename, at.Line+1)] = true
			}
		}
	}

	out := *pass
	report := pass.Report

	out.Report = func(d analysis.Diagnostic) {
		at := pass.Fset.Position(d.Pos)
		if !ignored[fmt.Sprintf("%s:%d", at.Filename, at.Line)] {
			report(d)
		}
	}

	return &out
}

// findKeys records the key that each model's Configure method sets.
func findKeys(pass *analysis.Pass) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Configure" || fn.Recv == nil || fn.Body == nil {
				continue
			}

			model := receiverType(pass, fn)
			if model == nil {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				if field, ok := setsKey(pass, call); ok {
					pass.ExportObjectFact(model.Obj(), &keyFact{Field: field})
				}

				return true
			})
		}
	}
}

// receiverType returns the named type a method is declared on.
func receiverType(pass *analysis.Pass, fn *ast.FuncDecl) *types.Named {
	object, ok := pass.TypesInfo.Defs[fn.Name].(*types.Func)
	if !ok {
		return nil
	}

	recv := object.Type().(*types.Signature).Recv()
	if recv == nil {
		return nil
	}

	t := types.Unalias(recv.Type())
	if pointer, ok := t.(*types.Pointer); ok {
		t = types.Unalias(pointer.Elem())
	}

	named, _ := t.(*types.Named)

	return named
}

// setsKey reports whether call sets a primary key, and of which field, if Find
// can take it.
func setsKey(pass *analysis.Pass, call *ast.CallExpr) (string, bool) {
	selector, ok := ast.Unparen(indexed(call.Fun)).(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	object, ok := pass.TypesInfo.Uses[selector.Sel].(*types.Func)
	if !ok || object.Pkg() == nil || object.Pkg().Path() != antPath {
		return "", false
	}

	switch object.Name() {
	case "Key":
		if len(call.Args) == 1 {
			return fieldName(call.Args[0]), true
		}
	case "GenerateKey":
		if len(call.Args) == 3 {
			return fieldName(call.Args[1]), true
		}
	case "CompositeKey", "NoKey":
		return "", true
	}

	return "", false
}

// indexed strips the type arguments from a generic function or method value.
func indexed(fun ast.Expr) ast.Expr {
	switch index := ast.Unparen(fun).(type) {
	case *ast.IndexExpr:
		return index.X
	case *ast.IndexListExpr:
		return index.X
	}

	return fun
}

// fieldName returns F of &X.F.
func fieldName(arg ast.Expr) string {
	unary, ok := ast.Unparen(arg).(*ast.UnaryExpr)
	if !ok {
		return ""
	}

	selector, ok := ast.Unparen(unary.X).(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	return selector.Sel.Name
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

// modelOf names the model a probe points to. An alias, such as
// type User = fixture.User, names the same model as what it stands for.
func modelOf(t types.Type) string {
	pointer, ok := types.Unalias(t).(*types.Pointer)
	if !ok {
		return ""
	}

	return types.TypeString(types.Unalias(pointer.Elem()), nil)
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
	if model := queryArg(t); model != nil {
		return types.TypeString(model, nil)
	}

	return ""
}

// queryArg returns T of a *Query[T] or *Table[T], unaliased, or nil.
func queryArg(t types.Type) types.Type {
	if t == nil {
		return nil
	}

	t = types.Unalias(t)
	if pointer, ok := t.(*types.Pointer); ok {
		t = types.Unalias(pointer.Elem())
	}

	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != antPath {
		return nil
	}

	if name := named.Obj().
		Name(); (name != "Query" && name != "Table") ||
		named.TypeArgs().Len() != 1 {
		return nil
	}

	return types.Unalias(named.TypeArgs().At(0))
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

// checkWhere reports a condition on a model other than the query's. A subquery's
// conditions may name its outer query's models, so a query is judged only where
// it is plainly not one: when its chain ends in a call that runs it, such as All,
// or when it sits inside an outer query, whose models it may also name.
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

	allowed := []string{model}

	if !c.runs(stack) {
		outer := c.outerModels(stack)
		if len(outer) == 0 {
			return
		}

		allowed = append(allowed, outer...)
	}

	for _, arg := range call.Args {
		condition, ok := ast.Unparen(arg).(*ast.CallExpr)
		if !ok || !singleModel[calleeName(pass, condition)] || len(condition.Args) == 0 {
			continue
		}

		if _, probe, _ := c.probeOf(condition.Args[0]); probe != "" &&
			!slices.Contains(allowed, probe) {
			pass.Reportf(
				condition.Pos(),
				"condition on %s in a query of %s",
				short(probe),
				short(model),
			)
		}
	}
}

// runs reports whether the chain that the innermost node of stack starts ends in
// a call that runs the query, which leaves the query types: All, Count, Update.
func (c *checker) runs(stack []ast.Node) bool {
	node := stack[len(stack)-1]

	for i := len(stack) - 2; i >= 1; i -= 2 {
		selector, ok := stack[i].(*ast.SelectorExpr)
		if !ok || selector.X != node {
			return false
		}

		next, ok := stack[i-1].(*ast.CallExpr)
		if !ok || next.Fun != selector {
			return false
		}

		if queryArg(c.pass.TypesInfo.TypeOf(next)) == nil {
			return true
		}

		node = next
	}

	return false
}

// outerModels returns the models of the queries whose Where calls enclose the
// innermost node of stack.
func (c *checker) outerModels(stack []ast.Node) []string {
	var out []string

	for _, n := range stack[:len(stack)-1] {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			continue
		}

		selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Where" {
			continue
		}

		if model := queryModel(c.pass.TypesInfo.TypeOf(selector.X)); model != "" {
			out = append(out, model)
		}
	}

	return out
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
// field, or the single field its Configure method makes the key.
func checkFind(pass *analysis.Pass, call *ast.CallExpr) {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Find" || len(call.Args) != 2 {
		return
	}

	model := queryArg(pass.TypesInfo.TypeOf(selector.X))
	if model == nil {
		return
	}

	name := "ID"

	if named, ok := model.(*types.Named); ok {
		fact := new(keyFact)
		if pass.ImportObjectFact(named.Obj(), fact) {
			name = fact.Field
		}
	}

	if name == "" {
		return
	}

	object, _, _ := types.LookupFieldOrMethod(model, true, pass.Pkg, name)

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
