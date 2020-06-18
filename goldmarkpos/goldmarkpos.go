package main

import (
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

const input = `
[link1](/abc)
[link2][1]
[link2]

![image](def.png)

[1]: /ghi
[link2]: /jkl
`

func main() {
	source := []byte(input)
	root := goldmark.DefaultParser().Parse(text.NewReader(source))
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			walk(source, n)
		}
		return ast.WalkContinue, nil
	})
}

func walk(source []byte, n ast.Node) {
	var dest string
	switch n := n.(type) {
	case *ast.Link:
		dest = string(n.Destination)
	case *ast.Image:
		dest = string(n.Destination)
	default:
		return
	}
	line, col := lineColForNode(source, n)
	fmt.Printf("%d:%d: %s\n", line, col, dest)
}

func lineColForNode(source []byte, node ast.Node) (line, col int) {
	line, col = -1, -1
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if text, ok := n.(*ast.Text); ok {
				line, col = lineColFromOffset(source, text.Segment.Start-1)
				return ast.WalkStop, nil
			}
		}
		return ast.WalkContinue, nil
	})
	return line, col
}

func lineColFromOffset(source []byte, offset int) (line, col int) {
	line = 1
	col = 1
	for i := 0; i < offset; i++ {
		if source[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}
