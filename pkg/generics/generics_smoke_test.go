package generics

import (
	"errors"
	"testing"
)

func TestResultSmoke(t *testing.T) {
	r := Ok(42)
	if !r.IsOk() {
		t.Error("expected ok")
	}
	if r.IsErr() {
		t.Error("expected not err")
	}
	v, err := r.Unwrap()
	if v != 42 || err != nil {
		t.Errorf("Unwrap = (%d, %v), want (42, nil)", v, err)
	}

	r2 := Err[int](errors.New("test error"))
	if r2.IsOk() {
		t.Error("expected not ok")
	}

	wr := WrapResult(10, nil)
	if wr.Val != 10 {
		t.Error("expected 10")
	}
}

func TestOptionalSmoke(t *testing.T) {
	o := Some(100)
	if !o.IsSome() {
		t.Error("expected present")
	}
	v, ok := o.Unwrap()
	if v != 100 || !ok {
		t.Errorf("Unwrap = (%d, %v), want (100, true)", v, ok)
	}

	o2 := None[int]()
	if o2.IsSome() {
		t.Error("expected not present")
	}
}

func TestSetSmoke(t *testing.T) {
	s := NewSet(1, 2, 3)
	s.Add(1)
	if s.Len() != 3 {
		t.Errorf("expected len 3, got %d", s.Len())
	}
	if !s.Has(1) {
		t.Error("expected contains 1")
	}
	if s.Has(4) {
		t.Error("expected not contains 4")
	}
	s.Remove(1)
	if s.Has(1) {
		t.Error("expected not contain 1 after remove")
	}
}

func TestStackSmoke(t *testing.T) {
	s := NewStack[string]()
	s.Push("a")
	s.Push("b")
	if s.Len() != 2 {
		t.Errorf("expected len 2, got %d", s.Len())
	}
	v, ok := s.Pop()
	if !ok || v != "b" {
		t.Errorf("Pop = (%q, %v), want (b, true)", v, ok)
	}
	v, ok = s.Pop()
	if !ok || v != "a" {
		t.Errorf("Pop = (%q, %v), want (a, true)", v, ok)
	}
	_, ok = s.Pop()
	if ok {
		t.Error("expected pop on empty stack to return false")
	}
	if !s.IsEmpty() {
		t.Error("expected empty after popping all")
	}
}
