// Package pointer provides utliities for pointer types.
package pointer

// From returns a pointer to the given value.
func From[T any](v T) *T {
	return &v
}

// OrZero returns the value of the given pointer or the zero value of the type if the pointer is nil.
//
// Example:
//
//	var n *int
//	pointer.OrZero(n) == 0
func OrZero[T any](v *T) T {
	var zero T
	if v == nil {
		return zero
	}
	return *v
}
