package main

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/ua-parser/uap-go/uaparser"
)

func BenchmarkUAP(b *testing.B) {
	d, err := ioutil.ReadFile("useragents.txt")
	if err != nil {
		b.Fatal(err)
	}
	ds := bytes.Split(d, []byte("\n"))
	uas := make([]string, len(ds))
	for i := range ds {
		uas[i] = string(ds[i])
	}
	parser := uaparser.NewFromSaved()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = parser.ParseDevice(uas[i%len(uas)])
	}
}
