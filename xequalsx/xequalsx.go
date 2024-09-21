package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var Analyzer = &analysis.Analyzer{
	Name: "xequalsx",
	Doc:  "Find 'x := x' declarations unneeded in Go >= 1.22",
	Run:  run,
}

type inspector struct {
	pass  *analysis.Pass
	stack []*scope
}

type scope struct {
	loopVars  []*loopVar
	funcDepth int
}

type loopVar struct {
	name  string
	obj   types.Object
	assns []*assignment
}

type assignment struct {
	lhs       *ast.Ident
	lhsObj    types.Object
	funcDepth int
	sameName  bool
	captured  bool
}

func run(pass *analysis.Pass) (any, error) {
	in := &inspector{pass: pass}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			in.inspect(n)
			return true
		})
	}
	return nil, nil
}

// The basic idea as we walk the AST:
//
// * When we find a for loop, note the loop vars.
// * When we find an assignment of the form ident := ident, check if the RHS is
//   one of the loop vars in an enclosing scope. If so, append it to that loop
//   var's list of associated assignments.
// * When we find any function literal, increment the funcDepth.
// * When we find any ident, check if it resolves to the LHS of one of the
//   assignments we've associated with the loopvars in one of the enclosing
//   scopes. If it does, and if the funcDepth for that assignment is different
//   from the current funcDepth, then that assignment's LHS is captured by a
//   func literal.
// * Print out warnings for all captured assignments.

func (in *inspector) inspect(n ast.Node) {
	if n == nil {
		in.popScope()
		return
	}
	s := new(scope)
	if len(in.stack) > 0 {
		s.funcDepth = in.curScope().funcDepth
	}
	in.stack = append(in.stack, s)

	switch n := n.(type) {
	case *ast.RangeStmt:
		in.addVar(n.Key)
		in.addVar(n.Value)
	case *ast.ForStmt:
		switch post := n.Post.(type) {
		case *ast.AssignStmt:
			for _, lhs := range post.Lhs {
				in.addVar(lhs)
			}
		case *ast.IncDecStmt:
			in.addVar(post.X)
		}
	case *ast.AssignStmt:
		if len(n.Lhs) != len(n.Rhs) {
			// Not a simple var assignment. (Probably 'x, err := f()'.)
			break
		}
		for i, lhs := range n.Lhs {
			lhsIdent, ok := lhs.(*ast.Ident)
			if !ok {
				break
			}
			rhs := n.Rhs[i]
			rhsIdent, ok := rhs.(*ast.Ident)
			if !ok {
				break
			}
			in.inspectIdentAssign(lhsIdent, rhsIdent)
		}
	case *ast.FuncLit:
		in.curScope().funcDepth++
	case *ast.Ident:
		in.inspectIdent(n)
	}
}

func (in *inspector) curScope() *scope {
	return in.stack[len(in.stack)-1]
}

func (in *inspector) addVar(expr ast.Expr) {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return
	}
	if obj := in.pass.TypesInfo.ObjectOf(ident); obj != nil {
		s := in.curScope()
		s.loopVars = append(s.loopVars, &loopVar{
			name: ident.Name,
			obj:  obj,
		})
	}
}

func (in *inspector) inspectIdentAssign(lhs, rhs *ast.Ident) {
	if rhsObj := in.pass.TypesInfo.Uses[rhs]; rhsObj != nil {
		for _, s := range in.stack {
			for _, v := range s.loopVars {
				if rhsObj == v.obj {
					if lhsObj := in.pass.TypesInfo.ObjectOf(lhs); lhsObj != nil {
						v.assns = append(v.assns, &assignment{
							lhs:       lhs,
							lhsObj:    lhsObj,
							sameName:  lhs.Name == rhs.Name,
							funcDepth: s.funcDepth,
						})
						return
					}
				}
			}
		}
	}
	if lhs.Name == rhs.Name {
		in.pass.ReportRangef(lhs, "same-name assignment does not reference loop var (mistake?)")
	}
}

func (in *inspector) inspectIdent(ident *ast.Ident) {
	obj := in.pass.TypesInfo.Uses[ident]
	if obj == nil {
		return
	}
	funcDepth := in.curScope().funcDepth
	for _, s := range in.stack {
		for _, v := range s.loopVars {
			for _, assn := range v.assns {
				if obj == assn.lhsObj && funcDepth > assn.funcDepth {
					assn.captured = true
					return
				}
			}
		}
	}
}

func (in *inspector) popScope() {
	s := in.curScope()
	in.stack = in.stack[:len(in.stack)-1]

	for _, v := range s.loopVars {
		for _, assn := range v.assns {
			if assn.captured {
				in.pass.ReportRangef(assn.lhs, "possibly-unnecessary assignment in Go >= 1.22")
			} else if assn.sameName {
				in.pass.ReportRangef(assn.lhs, "loop var re-declaration is not captured by func literal (mistake?)")
			}
		}
	}
}

func main() {
	singlechecker.Main(Analyzer)
}
