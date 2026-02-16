package main

import (
	"fmt"

	cloudfoundry "code.cloudfoundry.org/bytefmt"
	"github.com/alecthomas/units"
	allenai "github.com/allenai/bytefmt"
	"github.com/dustin/go-humanize"
	"github.com/inhies/go-bytesize"
)

func main() {
	for _, s := range []string{
		"1.23456789123 EB",
		"1.234GB",
		"1.234 MiB",
	} {
		testHumanizeParse(s)
		testCloudfoundryParse(s)
		testUnitsParse(s)
		testBytesizeParse(s)
		testAllenaiParse(s)
	}

	fmt.Println("---------------------------------------------------")

	for _, n := range []int64{
		999_999,
		1_000_000,
	} {
		testHumanizeFormat(n)
		testCloudfoundryFormat(n)
		testUnitsFormat(n)
		testBytesizeFormat(n)
		testAllenaiFormat(n)
	}
}

func testHumanizeParse(s string) {
	n, err := humanize.ParseBytes(s)
	if err != nil {
		fmt.Printf("humanize parse(%q): %s\n", s, err)
		return
	}
	fmt.Printf("humanize parse(%q): %d\n", s, n)
}

func testHumanizeFormat(n int64) {
	s := humanize.Bytes(uint64(n))
	fmt.Printf("humanize format %d: %s\n", n, s)
}

func testCloudfoundryParse(s string) {
	n, err := cloudfoundry.ToBytes(s)
	if err != nil {
		fmt.Printf("cloudfoundry bytefmt parse(%q): %s\n", s, err)
		return
	}
	fmt.Printf("cloudfoundry bytefmt parse(%q): %d\n", s, n)
}

func testCloudfoundryFormat(n int64) {
	s := cloudfoundry.ByteSize(uint64(n))
	fmt.Printf("cloudfoundry format %d: %s\n", n, s)
}

func testUnitsParse(s string) {
	n, err := units.ParseStrictBytes(s)
	if err != nil {
		fmt.Printf("units parse(%q): %s\n", s, err)
		return
	}
	fmt.Printf("units parse(%q): %d\n", s, n)
}

func testUnitsFormat(n int64) {
	s := units.MetricBytes(n)
	fmt.Printf("units format %d: %s\n", n, s.String())
}

func testBytesizeParse(s string) {
	n, err := bytesize.Parse(s)
	if err != nil {
		fmt.Printf("bytesize parse(%q): %s\n", s, err)
		return
	}
	fmt.Printf("bytesize parse(%q): %d\n", s, n)
}

func testBytesizeFormat(n int64) {
	s := bytesize.ByteSize(n)
	fmt.Printf("bytesize format %d: %s\n", n, s)
}

func testAllenaiParse(s string) {
	n, err := allenai.Parse(s)
	if err != nil {
		fmt.Printf("allenai bytefmt parse(%q): %s\n", s, err)
		return
	}
	fmt.Printf("allenai bytefmt parse(%q): %d\n", s, n.Int64())
}

func testAllenaiFormat(n int64) {
	s := allenai.New(n, allenai.Metric)
	fmt.Printf("allenai format %d: %s\n", n, s.String())
}
