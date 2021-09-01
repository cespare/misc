package main

import (
	"github.com/cespare/misc/prototest/x"
	"github.com/kr/pretty"
	"google.golang.org/protobuf/proto"
)

func main() {
	_ = make(map[x.Foo]struct{})
	foo := &x.Foo{Bar: 3}
	pretty.Println(proto.Equal(foo, &x.Foo{}))
}
