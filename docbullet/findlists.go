// +build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	log.SetFlags(0)
	if err := filepath.Walk(os.Args[1], walk); err != nil {
		log.Fatal(err)
	}
}

func walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		if info.Name() == "testdata" {
			return filepath.SkipDir
		}
		if err := processDir(path); err != nil {
			return err
		}
	}
	return nil
}

func processDir(dir string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, noTests, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, pkg := range sortPackages(pkgs) {
		p := doc.New(pkg, "", 0)
		process := func(doc, whence string) {
			if interesting(doc) {
				fmt.Println(whence)
				//fmt.Println(doc)
				//fmt.Println()
			}
		}
		process(p.Doc, p.Name)
		for _, c := range p.Consts {
			process(c.Doc, p.Name+".{"+strings.Join(c.Names, ", ")+"}")
		}
		for _, v := range p.Vars {
			process(v.Doc, p.Name+".{"+strings.Join(v.Names, ", ")+"}")
		}
		for _, t := range p.Types {
			process(t.Doc, p.Name+"."+t.Name)
		}
		for _, f := range p.Funcs {
			process(f.Doc, p.Name+"."+f.Name)
		}
	}
	return nil
}

func sortPackages(m map[string]*ast.Package) []*ast.Package {
	var names []string
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	var pkgs []*ast.Package
	for _, name := range names {
		if name == "main" {
			continue
		}
		pkgs = append(pkgs, m[name])
	}
	return pkgs
}

func interesting(doc string) bool {
	typ := findLists(doc)
	//return typ > 0
	return typ&listTypeMalformed > 0
}

var listRegexps = []struct {
	re  *regexp.Regexp
	typ listType
}{
	{regexp.MustCompile(`^\s*- `), listTypeHyphen},
	{regexp.MustCompile(`^\s*\+ `), listTypePlus},
	{regexp.MustCompile(`^\s*\* `), listTypeStar},
	{regexp.MustCompile(`^\s*• `), listTypeUnicode},
	{regexp.MustCompile(`^\s*[0-9][)\.] `), listTypeNumber},
	{regexp.MustCompile(`^[-+*] `), listTypeMalformed},
}

func findLists(doc string) listType {
	var typ listType
	for _, line := range strings.Split(doc, "\n") {
		for _, lr := range listRegexps {
			if lr.re.MatchString(line) {
				typ |= lr.typ
			}
		}
	}
	return typ
}

type listType uint

const (
	listTypeHyphen listType = 1 << iota
	listTypeStar
	listTypePlus
	listTypeNumber
	listTypeUnicode
	listTypeMalformed
)

func noTests(info os.FileInfo) bool {
	return !strings.HasSuffix(info.Name(), "_test.go")
}
