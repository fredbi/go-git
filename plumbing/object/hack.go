package object

import "unsafe"

// internalString representation of a string by the golang runtime
type internalString struct {
	Data unsafe.Pointer
	Len  int
}

// hackZeroAlloc reuses a common hack found in the standard library
// to avoid allocating the underlying bytes of a string when converting.
//
// This assumes that the caller does not use the returned []byte slices after
// having relinquished the input string to the garbage collector.
func hackZeroAlloc(s string) []byte {
	addr := (*internalString)(unsafe.Pointer(&s)).Data

	return unsafe.Slice((*byte)(addr), len(s))
}
