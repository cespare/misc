package main

import (
	"errors"
	"reflect"
	"strconv"
)

type Node struct {
	Data int
	Next *Node
}

func ToSlice(head *Node) []int {
	var s []int
	for node := head; node != nil; node = node.Next {
		s = append(s, node.Data)
	}
	return s
}

func FromSlice(s ...int) *Node {
	var l *Node
	for i := len(s) - 1; i >= 0; i-- {
		Push(&l, s[i])
	}
	return l
}

func Equal(head0, head1 *Node) bool {
	s0, s1 := ToSlice(head0), ToSlice(head1)
	return reflect.DeepEqual(s0, s1)
}

func Length(head *Node) int {
	n := 0
	for node := head; node != nil; node = node.Next {
		n++
	}
	return n
}

func BuildOneTwoThree() *Node {
	return FromSlice(1, 2, 3)
}

func Push(headRef **Node, newData int) {
	*headRef = &Node{
		Data: newData,
		Next: *headRef,
	}
}

func String(headRef **Node) string {
	s := "{"
	first := true
	for node := *headRef; node != nil; node = node.Next {
		if !first {
			s += ", "
		}
		first = false
		s += strconv.Itoa(node.Data)
	}
	s += "}"
	return s
}

func Count(head *Node, searchFor int) int {
	var n int
	for node := head; node != nil; node = node.Next {
		if node.Data == searchFor {
			n++
		}
	}
	return n
}

var errOutOfRange = errors.New("index out of range")

func GetNth(head *Node, index int) int {
	node := head
	for i := 0; i < index; i++ {
		if node == nil {
			panic(errOutOfRange)
		}
		node = node.Next
	}
	if node == nil {
		panic(errOutOfRange)
	}
	return node.Data
}

func DeleteList(headRef **Node) {
	*headRef = nil
}

var errPopEmpty = errors.New("Pop called on empty list")

func Pop(headRef **Node) int {
	if *headRef == nil {
		panic(errPopEmpty)
	}
	d := (*headRef).Data
	*headRef = (*headRef).Next
	return d
}

func InsertNth(headRef **Node, index int, data int) {
	ref := headRef
	for i := 0; i < index; i++ {
		if *ref == nil {
			panic(errOutOfRange)
		}
		ref = &(*ref).Next
	}
	*ref = &Node{
		Data: data,
		Next: *ref,
	}
}

func SortedInsert(headRef **Node, newNode *Node) {
	ref := headRef
	for *ref != nil && (*ref).Data < newNode.Data {
		ref = &(*ref).Next
	}
	newNode.Next = *ref
	*ref = newNode
}

func InsertSort(headRef **Node) {
	var l *Node
	node := *headRef
	for node != nil {
		next := node.Next
		SortedInsert(&l, node)
		node = next
	}
	*headRef = l
}

func Append(aRef, bRef **Node) {
	for *aRef != nil {
		*aRef = (*aRef).Next
	}
	*aRef = *bRef
	*bRef = nil
}

func FrontBackSplit(source *Node, frontRef, backRef **Node) {
}

func main() {
}
