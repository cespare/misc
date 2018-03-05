// Package i128 implements a 128-bit integer type.
//
// NOTE: This is the start of some code that I wrote down in a few minutes. It
// is probably wrong and is untested.
package i128

// Int128 represents a 128-bit signed integer.
type Int128 struct {
	lo int64
	hi int64
}

// And computes i & j.
func (i Int128) And(j Int128) Int128 {
	return Int128{
		lo: i.lo & j.lo,
		hi: i.hi & j.hi,
	}
}

// Or computes i | j.
func (i Int128) Or(j Int128) Int128 {
	return Int128{
		lo: i.lo | j.lo,
		hi: i.hi | j.hi,
	}
}

// Xor computes i ^ j.
func (i Int128) Xor(j Int128) Int128 {
	return Int128{
		lo: i.lo ^ j.lo,
		hi: i.hi ^ j.hi,
	}
}

// AndNot computes i &^ j.
func (i Int128) AndNot(j Int128) Int128 {
	return Int128{
		lo: i.lo &^ j.lo,
		hi: i.hi &^ j.hi,
	}
}

// Add computes i + j.
func (i Int128) Add(j Int128) Int128 {
	k := Int128{
		lo: i.lo + j.lo,
		hi: i.hi + j.hi,
	}
	if (k.lo < i.lo) != (j.lo < 0) {
		k.hi++
	}
	return k
}

// Mul computes i * j.
func (i Int128) Mul(j Int128) Int128 {
	panic("unimplemented")
}

// Div computes i / j.
func (i Int128) Div(j Int128) Int128 {
	panic("unimplemented")
}

// Rem computes i % j.
func (i Int128) Rem(j Int128) Int128 {
	panic("unimplemented")
}

// Neg computes -i.
func (i Int128) Neg() Int128 {
	panic("unimplemented")
}

// Comp computes ^i.
func (i Int128) Comp() Int128 {
	panic("unimplemented")
}

// Lsh computes i << n.
func (i Int128) Lsh(n uint) Int128 {
	panic("unimplemented")
}

// Rsh computes i >> n.
func (i Int128) Rsh(n uint) Int128 {
	panic("unimplemented")
}

// Gt computes i > j.
func (i Int128) Gt(j Int128) bool {
	panic("unimplemented")
}

// Lt computes i < j.
func (i Int128) Lt(j Int128) bool {
	panic("unimplemented")
}

// Geq computes i >= j.
func (i Int128) Geq(j Int128) bool {
	panic("unimplemented")
}

// Leq computes i <= j.
func (i Int128) Leq(j Int128) bool {
	panic("unimplemented")
}
