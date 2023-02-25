package main

import (
	"fmt"
	"log"

	"golang.org/x/tools/go/packages"
)

func main() {
	files := []string{
		"file=columns/columns.go",
	}
	config := &packages.Config{
		Mode:  packages.NeedName | packages.NeedImports,
		Dir:   ".",
		Tests: true,
	}
	pkgs, err := packages.Load(config, files...)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("packages.Load returned %d packages\n", len(pkgs))
	for _, pkg := range pkgs {
		fmt.Println(pkg)
	}
}
