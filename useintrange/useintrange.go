package main

import (
	"flag"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"
	"strconv"

	"golang.org/x/tools/go/packages"
	"rsc.io/edit"
)

var verbose = flag.Bool("v", false, "Operate in verbose mode")

func main() {
	log.SetFlags(0)
	flag.Parse()

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedCompiledGoFiles |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax,
	}
	pkgs, err := packages.Load(cfg, flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}
	var numFixes, numFiles int
	for _, pkg := range pkgs {
		if *verbose {
			log.Printf("Processing %s...", pkg.PkgPath)
		}
		nfx, nfi := fixPackage(pkg)
		numFixes += nfx
		numFiles += nfi
	}
	log.Printf("Fixed %d for statements across %d files", numFixes, numFiles)
}

func fixPackage(pkg *packages.Package) (numFixes, numFiles int) {
	if len(pkg.CompiledGoFiles) != len(pkg.Syntax) {
		log.Fatalf(
			"len(CompiledGoFiles)=%d; len(Syntax)=%d",
			len(pkg.CompiledGoFiles), len(pkg.Syntax),
		)
	}
	for i, file := range pkg.Syntax {
		name := pkg.CompiledGoFiles[i]
		targets := locateTargets(pkg, file)
		if len(targets) == 0 {
			continue
		}
		numFixes += len(targets)
		numFiles++
		if *verbose {
			log.Printf("% 4d %s", len(targets), name)
		}
		if err := fixTargets(pkg, name, targets); err != nil {
			log.Fatalf("Error editing %s: %s", name, err)
		}
	}
	return numFixes, numFiles
}

type target struct {
	stmt        *ast.ForStmt
	bodyUsesVar bool
}

func locateTargets(pkg *packages.Package, file *ast.File) []target {
	var targets []target
	ast.Inspect(file, func(n ast.Node) bool {
		stmt, ok := n.(*ast.ForStmt)
		if !ok {
			return true
		}
		ident := stmtCanUseRange(pkg, stmt)
		if ident == nil {
			return true
		}
		targ := target{
			stmt:        stmt,
			bodyUsesVar: bodyUses(pkg, stmt.Body, ident),
		}
		targets = append(targets, targ)
		return true
	})
	return targets
}

func stmtCanUseRange(pkg *packages.Package, stmt *ast.ForStmt) *ast.Ident {
	if stmt.Init == nil || stmt.Cond == nil || stmt.Post == nil {
		return nil
	}
	ident := initIsSimpleDecl(pkg, stmt.Init)
	if ident == nil {
		return nil
	}
	obj := pkg.TypesInfo.Defs[ident]
	if !condIsSimpleLessThan(pkg, stmt.Cond, obj) {
		return nil
	}
	if !postIsIncrement(pkg, stmt.Post, obj) {
		return nil
	}
	return ident
}

func initIsSimpleDecl(pkg *packages.Package, init ast.Stmt) *ast.Ident {
	assn, ok := init.(*ast.AssignStmt)
	if !ok {
		return nil
	}
	if assn.Tok != token.DEFINE {
		return nil
	}
	if len(assn.Lhs) != 1 || len(assn.Rhs) != 1 {
		return nil
	}
	ident, ok := assn.Lhs[0].(*ast.Ident)
	if !ok {
		return nil
	}
	if !isInt(pkg, assn.Rhs[0], 0) {
		return nil
	}
	return ident
}

func isInt(pkg *packages.Package, expr ast.Expr, val int64) bool {
	switch expr := expr.(type) {
	case *ast.BasicLit:
		return isLiteralInt(expr, val)
	case *ast.CallExpr:
		if len(expr.Args) != 1 {
			return false
		}
		lit, ok := expr.Args[0].(*ast.BasicLit)
		if !ok {
			return false
		}
		if !isLiteralInt(lit, val) {
			return false
		}

		ident, ok := expr.Fun.(*ast.Ident)
		if !ok {
			return false
		}
		typ := pkg.TypesInfo.Uses[ident].Type().Underlying()
		basic, ok := typ.(*types.Basic)
		if !ok {
			return false
		}
		if basic.Info()&types.IsInteger == 0 {
			return false
		}
		return true
	}
	return false
}

func isLiteralInt(lit *ast.BasicLit, val int64) bool {
	if lit.Kind != token.INT {
		return false
	}
	n, err := strconv.ParseInt(lit.Value, 10, 64)
	if err != nil {
		return false
	}
	return n == val
}

func condIsSimpleLessThan(pkg *packages.Package, expr ast.Expr, varObj types.Object) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	if bin.Op != token.LSS {
		return false
	}
	if !isMatchingIdent(pkg, bin.X, varObj) {
		return false
	}
	// Only accept conds with a constant value.
	return pkg.TypesInfo.Types[bin.Y].Value != nil
}

func postIsIncrement(pkg *packages.Package, stmt ast.Stmt, varObj types.Object) bool {
	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
			return false
		}
		if !isMatchingIdent(pkg, stmt.Lhs[0], varObj) {
			return false
		}
		bin, ok := stmt.Rhs[0].(*ast.BinaryExpr)
		if !ok {
			return false
		}
		if bin.Op != token.ADD {
			return false
		}
		if !isMatchingIdent(pkg, bin.X, varObj) {
			return false
		}
		basic, ok := bin.Y.(*ast.BasicLit)
		if !ok {
			return false
		}
		return isLiteralInt(basic, 1)
	case *ast.IncDecStmt:
		if stmt.Tok != token.INC {
			return false
		}
		return isMatchingIdent(pkg, stmt.X, varObj)
	}
	return false
}

func isMatchingIdent(pkg *packages.Package, expr ast.Expr, obj types.Object) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return pkg.TypesInfo.Uses[ident] == obj
}

func bodyUses(pkg *packages.Package, body *ast.BlockStmt, ident *ast.Ident) bool {
	obj := pkg.TypesInfo.Defs[ident]
	if obj == nil {
		panic("bad")
	}
	for _, stmt := range body.List {
		var seen bool
		ast.Inspect(stmt, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if pkg.TypesInfo.Uses[ident] == obj {
				seen = true
			}
			return true
		})
		if seen {
			return true
		}
	}
	return false
}

func fixTargets(pkg *packages.Package, filename string, targets []target) error {
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	buf := edit.NewBuffer(b)
	for _, targ := range targets {
		if targ.bodyUsesVar {
			// Convert
			//   for i := 0; i < N; i++ {
			// to
			//   for i := range N {
			initRHS := targ.stmt.Init.(*ast.AssignStmt).Rhs[0]
			initRHSPos := pkg.Fset.Position(initRHS.Pos())
			condRHS := targ.stmt.Cond.(*ast.BinaryExpr).Y
			condRHSPos := pkg.Fset.Position(condRHS.Pos())
			buf.Replace(initRHSPos.Offset, condRHSPos.Offset, "range ")

			condRHSPosEnd := pkg.Fset.Position(condRHS.End())
			openBracePos := pkg.Fset.Position(targ.stmt.Body.Lbrace)
			buf.Replace(condRHSPosEnd.Offset, openBracePos.Offset, " ")
		} else {
			// Convert
			//   for i := 0; i < N; i++ {
			// to
			//   for range N {
			initPos := pkg.Fset.Position(targ.stmt.Init.Pos())
			condRHS := targ.stmt.Cond.(*ast.BinaryExpr).Y
			condRHSPos := pkg.Fset.Position(condRHS.Pos())
			buf.Replace(initPos.Offset, condRHSPos.Offset, "range ")

			condRHSPosEnd := pkg.Fset.Position(condRHS.End())
			openBracePos := pkg.Fset.Position(targ.stmt.Body.Lbrace)
			buf.Replace(condRHSPosEnd.Offset, openBracePos.Offset, " ")
		}
	}

	return os.WriteFile(filename, buf.Bytes(), 0o644)
}
