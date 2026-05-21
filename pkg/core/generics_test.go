package core

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResultOk(t *testing.T) {
	r := Ok(42)
	assert.True(t, r.IsOk())
	assert.False(t, r.IsErr())
	assert.Equal(t, 42, r.Must())
}

func TestResultErr(t *testing.T) {
	r := Err[int](errors.New("failed"))
	assert.False(t, r.IsOk())
	assert.True(t, r.IsErr())
	assert.EqualError(t, r.Err, "failed")
}

func TestResultUnwrap(t *testing.T) {
	r := Ok("hello")
	val, err := r.Unwrap()
	assert.Equal(t, "hello", val)
	assert.NoError(t, err)

	e := Err[string](errors.New("boom"))
	_, err = e.Unwrap()
	assert.EqualError(t, err, "boom")
}

func TestResultMustPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	Err[int](errors.New("boom")).Must()
}

func TestResultOrElse(t *testing.T) {
	assert.Equal(t, 42, Ok(42).OrElse(0))
	assert.Equal(t, 0, Err[int](errors.New("boom")).OrElse(0))
}

func TestResultMap(t *testing.T) {
	r := Ok(21).Map(func(v int) int { return v * 2 })
	assert.Equal(t, 42, r.Must())

	e := Err[int](errors.New("boom")).Map(func(v int) int { return v * 2 })
	assert.True(t, e.IsErr())
}

func TestWrapResult(t *testing.T) {
	r := WrapResult(42, nil)
	assert.True(t, r.IsOk())

	e := WrapResult(0, errors.New("failed"))
	assert.True(t, e.IsErr())
}

func TestOptionalSome(t *testing.T) {
	o := Some("hello")
	assert.True(t, o.IsSome())
	assert.False(t, o.IsNone())
	assert.Equal(t, "hello", o.Must())
}

func TestOptionalNone(t *testing.T) {
	o := None[string]()
	assert.False(t, o.IsSome())
	assert.True(t, o.IsNone())
}

func TestOptionalMustPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	None[string]().Must()
}

func TestOptionalOrElse(t *testing.T) {
	assert.Equal(t, "hello", Some("hello").OrElse("default"))
	assert.Equal(t, "default", None[string]().OrElse("default"))
}

func TestOptionalUnwrap(t *testing.T) {
	v, ok := Some(42).Unwrap()
	assert.Equal(t, 42, v)
	assert.True(t, ok)

	_, ok = None[int]().Unwrap()
	assert.False(t, ok)
}

func TestSet(t *testing.T) {
	s := NewSet("a", "b", "c")
	assert.Equal(t, 3, s.Len())
	assert.True(t, s.Has("a"))
	assert.False(t, s.Has("d"))

	s.Add("d")
	assert.True(t, s.Has("d"))
	assert.Equal(t, 4, s.Len())

	s.Remove("a")
	assert.False(t, s.Has("a"))
	assert.Equal(t, 3, s.Len())

	items := s.Items()
	assert.Contains(t, items, "b")
	assert.Contains(t, items, "c")
	assert.Contains(t, items, "d")
}

func TestSetUnion(t *testing.T) {
	a := NewSet(1, 2, 3)
	b := NewSet(3, 4, 5)
	u := a.Union(b)
	assert.Equal(t, 5, u.Len())
	for _, v := range []int{1, 2, 3, 4, 5} {
		assert.True(t, u.Has(v))
	}
}

func TestSetIntersection(t *testing.T) {
	a := NewSet(1, 2, 3, 4)
	b := NewSet(3, 4, 5, 6)
	i := a.Intersection(b)
	assert.Equal(t, 2, i.Len())
	assert.True(t, i.Has(3))
	assert.True(t, i.Has(4))
}

func TestStack(t *testing.T) {
	s := NewStack[int]()
	assert.True(t, s.IsEmpty())

	s.Push(1)
	s.Push(2)
	s.Push(3)
	assert.Equal(t, 3, s.Len())

	v, ok := s.Peek()
	assert.True(t, ok)
	assert.Equal(t, 3, v)

	v, ok = s.Pop()
	assert.True(t, ok)
	assert.Equal(t, 3, v)
	assert.Equal(t, 2, s.Len())

	s.Pop()
	s.Pop()

	_, ok = s.Pop()
	assert.False(t, ok)
	assert.True(t, s.IsEmpty())
}
