package core

// Result represents a value-or-error outcome, inspired by Rust's Result<T, E>.
// It provides a type-safe alternative to Go's (T, error) return pattern,
// enabling functional-style error composition without type assertions.
//
// This is especially valuable for Web3 developers coming from TypeScript
// (Result from neverthrow/tRPC) or Rust, helping bridge the mental model
// to Go's explicit error handling.
//
// Example:
//
//	r := WrapResult(fetchBlock(100))
//	if r.IsErr() {
//	    return r.Err
//	}
//	block := r.Must() // panics if err != nil
//	next := r.Map(func(b *Block) uint64 { return b.Number })
type Result[T any] struct {
	Val T
	Err error
}

func Ok[T any](val T) Result[T] {
	return Result[T]{Val: val}
}

func Err[T any](err error) Result[T] {
	return Result[T]{Err: err}
}

func WrapResult[T any](val T, err error) Result[T] {
	return Result[T]{Val: val, Err: err}
}

func (r Result[T]) IsOk() bool  { return r.Err == nil }
func (r Result[T]) IsErr() bool { return r.Err != nil }

func (r Result[T]) Unwrap() (T, error) {
	return r.Val, r.Err
}

func (r Result[T]) Must() T {
	if r.Err != nil {
		panic("Result.Must called on error: " + r.Err.Error())
	}
	return r.Val
}

func (r Result[T]) OrElse(defaultVal T) T {
	if r.IsErr() {
		return defaultVal
	}
	return r.Val
}

func (r Result[T]) Map(fn func(T) T) Result[T] {
	if r.IsErr() {
		return r
	}
	return Ok(fn(r.Val))
}

// MapErr maps the error value using fn, leaving a successful value unchanged.
func (r Result[T]) MapErr(fn func(error) error) Result[T] {
	if r.IsErr() {
		return Err[T](fn(r.Err))
	}
	return r
}

// Optional represents a value that may or may not be present.
// It's the Go generics equivalent of Option<T> in Rust / Maybe in Haskell.
type Optional[T any] struct {
	val     T
	present bool
}

func Some[T any](val T) Optional[T] { return Optional[T]{val: val, present: true} }
func None[T any]() Optional[T]      { return Optional[T]{present: false} }

func (o Optional[T]) IsSome() bool      { return o.present }
func (o Optional[T]) IsNone() bool      { return !o.present }
func (o Optional[T]) Unwrap() (T, bool) { return o.val, o.present }
func (o Optional[T]) Must() T {
	if !o.present {
		panic("Optional.Must called on None")
	}
	return o.val
}

func (o Optional[T]) OrElse(defaultVal T) T {
	if o.present {
		return o.val
	}
	return defaultVal
}

// Set is a generic unordered collection of unique elements.
// It's useful for whitelist/blacklist checks, deduplication,
// and membership testing — common patterns in Web3 (e.g., token allowlists).
type Set[T comparable] map[T]struct{}

func NewSet[T comparable](items ...T) Set[T] {
	s := make(Set[T], len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

func (s Set[T]) Add(v T)      { s[v] = struct{}{} }
func (s Set[T]) Remove(v T)   { delete(s, v) }
func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }
func (s Set[T]) Len() int     { return len(s) }

func (s Set[T]) Items() []T {
	out := make([]T, 0, len(s))
	for v := range s {
		out = append(out, v)
	}
	return out
}

func (s Set[T]) Union(other Set[T]) Set[T] {
	out := make(Set[T], len(s)+len(other))
	for v := range s {
		out[v] = struct{}{}
	}
	for v := range other {
		out[v] = struct{}{}
	}
	return out
}

func (s Set[T]) Intersection(other Set[T]) Set[T] {
	out := make(Set[T])
	for v := range s {
		if other.Has(v) {
			out[v] = struct{}{}
		}
	}
	return out
}

// Stack is a generic LIFO (last-in-first-out) data structure.
// Useful for tracking call stacks, undo operations, and traversals.
type Stack[T any] struct {
	items []T
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{items: make([]T, 0)}
}

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }
func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, true
}

func (s *Stack[T]) Peek() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

func (s *Stack[T]) Len() int      { return len(s.items) }
func (s *Stack[T]) IsEmpty() bool { return len(s.items) == 0 }
