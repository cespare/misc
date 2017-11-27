package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func TestLength(t *testing.T) {
	got := Length(nil)
	if want := 0; got != want {
		t.Errorf("got %d; want %d", got, want)
	}
	got = Length(BuildOneTwoThree())
	if want := 3; got != want {
		t.Errorf("got %d; want %d", got, want)
	}
}

func TestCount(t *testing.T) {
	for _, tt := range []struct {
		node      *Node
		searchFor int
		want      int
	}{
		{nil, 3, 0},
		{BuildOneTwoThree(), 3, 1},
		{BuildOneTwoThree(), 5, 0},
	} {
		got := Count(tt.node, tt.searchFor)
		if got != tt.want {
			t.Errorf("Count(%s, %d): got %d; want %d",
				String(&tt.node), tt.searchFor, got, tt.want)
		}
	}
}

func TestGetNth(t *testing.T) {
	l := BuildOneTwoThree()
	got := GetNth(l, 2)
	if want := 3; got != want {
		t.Errorf("got %d; want %d", got, want)
	}
}

func TestPop(t *testing.T) {
	l := BuildOneTwoThree()
	for _, want := range []int{1, 2, 3} {
		got := Pop(&l)
		if got != want {
			t.Errorf("got %d; want %d", got, want)
		}
	}
}

func TestInsertNth(t *testing.T) {
	for _, tt := range []struct {
		l     *Node
		index int
		data  int
		want  []int
	}{
		{nil, 0, 3, []int{3}},
		{BuildOneTwoThree(), 0, 3, []int{3, 1, 2, 3}},
		{BuildOneTwoThree(), 1, 5, []int{1, 5, 2, 3}},
		{BuildOneTwoThree(), 2, 5, []int{1, 2, 5, 3}},
		{BuildOneTwoThree(), 3, 5, []int{1, 2, 3, 5}},
	} {
		s := String(&tt.l)
		InsertNth(&tt.l, tt.index, tt.data)
		got := ToSlice(tt.l)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("InsertNth(%s, %d, %d): got %v; want %v",
				s, tt.index, tt.data, got, tt.want)
		}

	}
}

func TestSortedInsert(t *testing.T) {
	for _, tt := range []struct {
		l    *Node
		data int
		want []int
	}{
		{nil, 3, []int{3}},
		{BuildOneTwoThree(), 0, []int{0, 1, 2, 3}},
		{BuildOneTwoThree(), 1, []int{1, 1, 2, 3}},
		{FromSlice(1, 3, 5), 2, []int{1, 2, 3, 5}},
		{FromSlice(1, 3, 5), 5, []int{1, 3, 5, 5}},
		{FromSlice(1, 3, 5), 6, []int{1, 3, 5, 6}},
	} {
		s := String(&tt.l)
		SortedInsert(&tt.l, &Node{Data: tt.data})
		got := ToSlice(tt.l)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("SortedInsert(%s, %d): got %v; want %v",
				s, tt.data, got, tt.want)
		}
	}
}

func testSort(t *testing.T, sortFunc func(**Node)) {
	for _, l := range []*Node{
		nil,
		FromSlice(1),
		FromSlice(1, 3, 2),
		FromSlice(1, 1, 1),
		FromSlice(3, 2, 1, 4, 5),
	} {
		s := String(&l)
		want := ToSlice(l)
		sort.Ints(want)
		sortFunc(&l)
		if got := ToSlice(l); !reflect.DeepEqual(got, want) {
			t.Errorf("sort(%s): got %v; want %v", s, got, want)
		}
	}
}

func TestInsertSort(t *testing.T) {
	testSort(t, InsertSort)
}

func TestAppend(t *testing.T) {
	for _, tt := range []struct {
		a []int
		b []int
	}{
		{nil, nil},
	} {
		al := FromSlice(tt.a...)
		bl := FromSlice(tt.b...)
		Append(&al, &bl)
		got := ToSlice(al)
		if want := append(tt.a, tt.b...); !reflect.DeepEqual(got, want) {
			t.Errorf("Append(%v, %v): got %v; want %v", tt.a, tt.b, got, want)
			continue
		}
		if bl != nil {
			t.Error("after Append, b was not null")
		}
	}
}

func TestFrontBackSplit(t *testing.T) {
	for _, tt := range []struct {
		source    []int
		wantFront []int
		wantBack  []int
	}{
		{nil, nil, nil},
		{[]int{1}, []int{1}, nil},
		{[]int{2, 3}, []int{2}, []int{3}},
		{[]int{4, 5, 6}, []int{4, 5}, []int{6}},
		{[]int{7, 8, 9, 10}, []int{7, 8}, []int{9, 10}},
	} {
		s := fmt.Sprintf("%v", tt.source)
		source := FromSlice(tt.source...)
		var front, back *Node
		FrontBackSplit(source, &front, &back)
		gotFront, gotBack := ToSlice(front), ToSlice(back)
		if !reflect.DeepEqual(gotFront, tt.wantFront) ||
			!reflect.DeepEqual(gotBack, tt.wantBack) {
			t.Errorf("FrontBackSplit(%s): got (%v, %v); want (%v, %v)",
				s, gotFront, gotBack, tt.wantFront, tt.wantBack)
		}
	}
}
