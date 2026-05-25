package core

import "github.com/rtcdance/chainpulse/pkg/generics"

type (
	Result[T any]     = generics.Result[T]
	Optional[T any]   = generics.Optional[T]
	Set[T comparable] = generics.Set[T]
	Stack[T any]      = generics.Stack[T]
)

func Ok[T any](val T) Result[T]                    { return generics.Ok(val) }
func Err[T any](err error) Result[T]               { return generics.Err[T](err) }
func WrapResult[T any](val T, err error) Result[T] { return generics.WrapResult(val, err) }

func Some[T any](val T) Optional[T] { return generics.Some(val) }
func None[T any]() Optional[T]      { return generics.None[T]() }

func NewSet[T comparable](items ...T) Set[T] { return generics.NewSet(items...) }
func NewStack[T any]() *Stack[T]             { return generics.NewStack[T]() }
