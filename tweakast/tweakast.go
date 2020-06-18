package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"os/exec"
	"strings"
)

const source = `package main

import "x"

func init() {
	x.F(&x.Y{
		F1: 3,
		// Comments should be preserved!
		F2: []string{
			"a",
			"b", // even here
			"c", // and here
		},
	})
}
`

func main() {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "demo.go", source, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	for _, decl := range file.Decls {
		replaceB(decl)
	}
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, file)

	// Re-format the source.
	cmd := exec.Command("gofmt")
	cmd.Stdin = &buf
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalln("Cannot run gofmt on new source:", err)
	}
}

func replaceB(n ast.Decl) {
	initDecl, ok := n.(*ast.FuncDecl)
	if !ok || initDecl.Name.Name != "init" {
		return
	}
	for _, stmt := range initDecl.Body.List {
		es, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := es.X.(*ast.CallExpr)
		if !ok || !isSelector(call.Fun, "x.F") {
			continue
		}
		if len(call.Args) != 1 {
			continue
		}
		unExp, ok := call.Args[0].(*ast.UnaryExpr)
		if !ok || unExp.Op != token.AND {
			continue
		}
		compLit, ok := unExp.X.(*ast.CompositeLit)
		if !ok || !isSelector(compLit.Type, "x.Y") {
			continue
		}
		for _, elt := range compLit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				break
			}
			if !isIdent(kv.Key, "F2") {
				continue
			}
			sliceLit, ok := kv.Value.(*ast.CompositeLit)
			if !ok || !isStringSliceType(sliceLit.Type) {
				continue
			}
			for _, elt := range sliceLit.Elts {
				lit, ok := elt.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				if lit.Value != `"b"` {
					continue
				}
				// Found the node we're interested in replacing.
				lit.Value = `"zzz"`
			}
		}
	}
}
func isIdent(n ast.Node, name string) bool {
	if id, ok := n.(*ast.Ident); ok {
		return id.Name == name
	}
	return false
}

func isSelector(n ast.Node, name string) bool {
	sel, ok := n.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	parts := strings.SplitN(name, ".", 2)
	if len(parts) < 2 {
		panic("bad selector name")
	}
	if sel.Sel.Name != parts[1] {
		return false
	}
	return isIdent(sel.X, parts[0])
}

func isStringSliceType(expr ast.Expr) bool {
	arr, ok := expr.(*ast.ArrayType)
	if !ok {
		return false
	}
	if arr.Len != nil {
		return false
	}
	return isIdent(arr.Elt, "string")
}
