package commons

import "github.com/helloworldpark/tickle-stock-watcher/logger"

// ReferenceCounting is an interface for reference counting structs
type ReferenceCounting interface {
	Retain()
	Release()
	Count() int
}

// Ref is a simple struct for reference counting.
// Will panic if overreleased.
type Ref struct {
	ref int
}

// Retain increases reference count of the struct.
func (r *Ref) Retain() {
	r.ref++
}

// Release decreases reference count of the struct.
// It is the holder's responsibility to make use of the reference count.
// It panics if the struct is overreleased.
func (r *Ref) Release() {
	if r.ref <= 0 {
		logger.Panic("Trying to overrelease a struct")
	}
	r.ref--
}

// Count returns the reference count of the struct.
func (r *Ref) Count() int {
	return r.ref
}
