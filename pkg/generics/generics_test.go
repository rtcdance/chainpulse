package generics

import (
	"errors"
	"fmt"
	"testing"
)

func TestResult_Ok(t *testing.T) {
	t.Parallel()
	r := Ok(42)
	if !r.IsOk() {
		t.Error("expected Ok result")
	}
	if r.IsErr() {
		t.Error("expected not Err")
	}
	v, err := r.Unwrap()
	if v != 42 {
		t.Errorf("expected 42, got %v", v)
	}
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestResult_Err(t *testing.T) {
	t.Parallel()
	r := Err[int](errors.New("boom"))
	if r.IsOk() {
		t.Error("expected not Ok")
	}
	if !r.IsErr() {
		t.Error("expected Err")
	}
	_, err := r.Unwrap()
	if err == nil {
		t.Error("expected error")
	}
}

func TestResult_WrapResult(t *testing.T) {
	t.Parallel()
	r := WrapResult(10, nil)
	if !r.IsOk() {
		t.Error("expected Ok")
	}

	r2 := WrapResult(0, errors.New("fail"))
	if !r2.IsErr() {
		t.Error("expected Err")
	}
}

func TestResult_Must_Panic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	Err[int](errors.New("boom")).Must()
}

func TestResult_Must_NoPanic(t *testing.T) {
	t.Parallel()
	v := Ok(99).Must()
	if v != 99 {
		t.Errorf("expected 99, got %v", v)
	}
}

func TestResult_OrElse(t *testing.T) {
	t.Parallel()
	if Ok(1).OrElse(2) != 1 {
		t.Error("expected 1")
	}
	if Err[int](errors.New("err")).OrElse(2) != 2 {
		t.Error("expected 2")
	}
}

func TestResult_Map(t *testing.T) {
	t.Parallel()
	r := Ok(5).Map(func(x int) int { return x * 2 })
	v, _ := r.Unwrap()
	if v != 10 {
		t.Errorf("expected 10, got %v", v)
	}

	r2 := Err[int](errors.New("err")).Map(func(x int) int { return x * 2 })
	if !r2.IsErr() {
		t.Error("expected Err after Map on error")
	}
}

func TestResult_MapErr(t *testing.T) {
	t.Parallel()
	original := errors.New("original")
	r := Err[int](original).MapErr(func(e error) error {
		return fmt.Errorf("wrapped: %w", e)
	})
	_, err := r.Unwrap()
	if err == nil {
		t.Error("expected error")
	}

	r2 := Ok(1).MapErr(func(e error) error { return errors.New("should not happen") })
	if !r2.IsOk() {
		t.Error("expected Ok after MapErr on Ok")
	}
}

func TestOptional_Some(t *testing.T) {
	t.Parallel()
	o := Some("hello")
	if !o.IsSome() {
		t.Error("expected Some")
	}
	if o.IsNone() {
		t.Error("expected not None")
	}
	v, ok := o.Unwrap()
	if !ok {
		t.Error("expected ok")
	}
	if v != "hello" {
		t.Errorf("expected hello, got %v", v)
	}
}

func TestOptional_None(t *testing.T) {
	t.Parallel()
	o := None[int]()
	if o.IsSome() {
		t.Error("expected not Some")
	}
	if !o.IsNone() {
		t.Error("expected None")
	}
	_, ok := o.Unwrap()
	if ok {
		t.Error("expected not ok")
	}
}

func TestOptional_Must_Panic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	None[int]().Must()
}

func TestOptional_Must_NoPanic(t *testing.T) {
	t.Parallel()
	v := Some(99).Must()
	if v != 99 {
		t.Errorf("expected 99, got %v", v)
	}
}

func TestOptional_OrElse(t *testing.T) {
	t.Parallel()
	if Some(1).OrElse(2) != 1 {
		t.Error("expected 1")
	}
	if None[int]().OrElse(2) != 2 {
		t.Error("expected 2")
	}
}

func TestSet_NewSet(t *testing.T) {
	t.Parallel()
	s := NewSet(1, 2, 3)
	if s.Len() != 3 {
		t.Errorf("expected len 3, got %d", s.Len())
	}
}

func TestSet_Add_Has_Remove(t *testing.T) {
	t.Parallel()
	s := NewSet[string]()
	s.Add("a")
	if !s.Has("a") {
		t.Error("expected to have a")
	}
	if s.Has("b") {
		t.Error("expected not to have b")
	}
	s.Remove("a")
	if s.Has("a") {
		t.Error("expected a removed")
	}
}

func TestSet_Items(t *testing.T) {
	t.Parallel()
	s := NewSet(1, 2)
	items := s.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	found := make(map[int]bool)
	for _, v := range items {
		found[v] = true
	}
	if !found[1] || !found[2] {
		t.Error("missing items")
	}
}

func TestSet_Union(t *testing.T) {
	t.Parallel()
	a := NewSet(1, 2)
	b := NewSet(2, 3)
	u := a.Union(b)
	if u.Len() != 3 {
		t.Errorf("expected 3, got %d", u.Len())
	}
	if !u.Has(1) || !u.Has(2) || !u.Has(3) {
		t.Error("missing union elements")
	}
}

func TestSet_Intersection(t *testing.T) {
	t.Parallel()
	a := NewSet(1, 2, 3)
	b := NewSet(2, 3, 4)
	i := a.Intersection(b)
	if i.Len() != 2 {
		t.Errorf("expected 2, got %d", i.Len())
	}
	if !i.Has(2) || !i.Has(3) {
		t.Error("missing intersection elements")
	}
}

func TestSet_Intersection_Empty(t *testing.T) {
	t.Parallel()
	a := NewSet(1, 2)
	b := NewSet(3, 4)
	i := a.Intersection(b)
	if i.Len() != 0 {
		t.Errorf("expected empty intersection, got %d", i.Len())
	}
}

func TestSet_Len(t *testing.T) {
	t.Parallel()
	s := NewSet[int]()
	if s.Len() != 0 {
		t.Errorf("expected 0, got %d", s.Len())
	}
	s.Add(1)
	if s.Len() != 1 {
		t.Errorf("expected 1, got %d", s.Len())
	}
}

func TestStack_Push_Pop(t *testing.T) {
	t.Parallel()
	s := NewStack[int]()
	s.Push(1)
	s.Push(2)

	v, ok := s.Pop()
	if !ok {
		t.Error("expected ok")
	}
	if v != 2 {
		t.Errorf("expected 2, got %v", v)
	}

	v, ok = s.Pop()
	if !ok {
		t.Error("expected ok")
	}
	if v != 1 {
		t.Errorf("expected 1, got %v", v)
	}
}

func TestStack_Pop_Empty(t *testing.T) {
	t.Parallel()
	s := NewStack[int]()
	_, ok := s.Pop()
	if ok {
		t.Error("expected not ok for empty pop")
	}
}

func TestStack_Peek(t *testing.T) {
	t.Parallel()
	s := NewStack[string]()
	s.Push("top")
	v, ok := s.Peek()
	if !ok {
		t.Error("expected ok")
	}
	if v != "top" {
		t.Errorf("expected top, got %v", v)
	}
	if s.Len() != 1 {
		t.Error("Peek should not remove element")
	}
}

func TestStack_Peek_Empty(t *testing.T) {
	t.Parallel()
	s := NewStack[int]()
	_, ok := s.Peek()
	if ok {
		t.Error("expected not ok for empty peek")
	}
}

func TestStack_Len_IsEmpty(t *testing.T) {
	t.Parallel()
	s := NewStack[int]()
	if !s.IsEmpty() {
		t.Error("expected empty")
	}
	if s.Len() != 0 {
		t.Errorf("expected len 0, got %d", s.Len())
	}
	s.Push(1)
	if s.IsEmpty() {
		t.Error("expected not empty")
	}
	if s.Len() != 1 {
		t.Errorf("expected len 1, got %d", s.Len())
	}
}

func TestSet_EmptyNewSet(t *testing.T) {
	t.Parallel()
	s := NewSet[string]()
	if s.Len() != 0 {
		t.Errorf("expected 0, got %d", s.Len())
	}
}

func TestStack_MultipleTypes(t *testing.T) {
	t.Parallel()
	type pair struct{ a, b int }
	s := NewStack[pair]()
	s.Push(pair{1, 2})
	v, ok := s.Peek()
	if !ok {
		t.Error("expected ok")
	}
	if v.a != 1 || v.b != 2 {
		t.Errorf("unexpected value: %+v", v)
	}
}
