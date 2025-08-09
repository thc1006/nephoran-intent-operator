//go:build go1.24

// Package generics provides type-safe generic implementations leveraging Go 1.24+ features
// for the Nephoran Intent Operator's telecommunications network orchestration platform.
package generics

import (
	"fmt"
)

// Result represents a value that may be successful (T) or contain an error (E).
// This follows the functional programming paradigm for explicit error handling
// with compile-time type safety and zero runtime overhead.
type Result[T, E any] struct {
	value T
	err   E
	ok    bool
}

// Ok creates a successful Result containing the given value.
func Ok[T, E any](value T) Result[T, E] {
	return Result[T, E]{
		value: value,
		ok:    true,
	}
}

// Err creates a failed Result containing the given error.
func Err[T, E any](err E) Result[T, E] {
	return Result[T, E]{
		err: err,
		ok:  false,
	}
}

// IsOk returns true if the Result contains a successful value.
func (r Result[T, E]) IsOk() bool {
	return r.ok
}

// IsErr returns true if the Result contains an error.
func (r Result[T, E]) IsErr() bool {
	return !r.ok
}

// Unwrap returns the value and error. Use when you need both values.
// The boolean indicates if the result was successful.
func (r Result[T, E]) Unwrap() (T, E, bool) {
	return r.value, r.err, r.ok
}

// Value returns the successful value or panics if the Result contains an error.
// Use only when you're certain the Result is successful.
func (r Result[T, E]) Value() T {
	if !r.ok {
		panic("called Value() on error Result")
	}
	return r.value
}

// Error returns the error or panics if the Result is successful.
// Use only when you're certain the Result contains an error.
func (r Result[T, E]) Error() E {
	if r.ok {
		panic("called Error() on successful Result")
	}
	return r.err
}

// ValueOr returns the value if successful, otherwise returns the default.
func (r Result[T, E]) ValueOr(defaultValue T) T {
	if r.ok {
		return r.value
	}
	return defaultValue
}

// Map transforms the value using the provided function if the Result is successful.
// Returns a new Result with the transformed value, or the original error.
func Map[T, U, E any](r Result[T, E], fn func(T) U) Result[U, E] {
	if !r.ok {
		return Err[U, E](r.err)
	}
	return Ok[U, E](fn(r.value))
}

// MapErr transforms the error using the provided function if the Result contains an error.
// Returns a new Result with the original value or the transformed error.
func MapErr[T, E, F any](r Result[T, E], fn func(E) F) Result[T, F] {
	if r.ok {
		return Ok[T, F](r.value)
	}
	return Err[T, F](fn(r.err))
}

// FlatMap chains Results together. If the current Result is successful,
// applies the function and returns its Result. Otherwise returns the error.
func FlatMap[T, U, E any](r Result[T, E], fn func(T) Result[U, E]) Result[U, E] {
	if !r.ok {
		return Err[U, E](r.err)
	}
	return fn(r.value)
}

// Filter returns the Result if the predicate returns true, otherwise returns an error.
func Filter[T, E any](r Result[T, E], predicate func(T) bool, errorOnFalse E) Result[T, E] {
	if !r.ok {
		return r
	}
	if !predicate(r.value) {
		return Err[T, E](errorOnFalse)
	}
	return r
}

// Option represents a value that may or may not exist.
// This provides type-safe nullable value handling with zero runtime overhead.
type Option[T any] struct {
	value T
	some  bool
}

// Some creates an Option containing the given value.
func Some[T any](value T) Option[T] {
	return Option[T]{
		value: value,
		some:  true,
	}
}

// None creates an empty Option.
func None[T any]() Option[T] {
	return Option[T]{
		some: false,
	}
}

// IsSome returns true if the Option contains a value.
func (o Option[T]) IsSome() bool {
	return o.some
}

// IsNone returns true if the Option is empty.
func (o Option[T]) IsNone() bool {
	return !o.some
}

// Unwrap returns the value and whether it exists.
func (o Option[T]) Unwrap() (T, bool) {
	return o.value, o.some
}

// Value returns the value or panics if the Option is empty.
func (o Option[T]) Value() T {
	if !o.some {
		panic("called Value() on None Option")
	}
	return o.value
}

// ValueOr returns the value if present, otherwise returns the default.
func (o Option[T]) ValueOr(defaultValue T) T {
	if o.some {
		return o.value
	}
	return defaultValue
}

// MapOption transforms the value using the provided function if present.
func MapOption[T, U any](o Option[T], fn func(T) U) Option[U] {
	if !o.some {
		return None[U]()
	}
	return Some(fn(o.value))
}

// FlatMapOption chains Options together.
func FlatMapOption[T, U any](o Option[T], fn func(T) Option[U]) Option[U] {
	if !o.some {
		return None[U]()
	}
	return fn(o.value)
}

// FilterOption returns the Option if the predicate returns true, otherwise None.
func FilterOption[T any](o Option[T], predicate func(T) bool) Option[T] {
	if !o.some || !predicate(o.value) {
		return None[T]()
	}
	return o
}

// ToResult converts an Option to a Result.
func (o Option[T]) ToResult(err error) Result[T, error] {
	if o.some {
		return Ok[T, error](o.value)
	}
	return Err[T, error](err)
}

// FromResult converts a Result to an Option, discarding any error.
func FromResult[T, E any](r Result[T, E]) Option[T] {
	if r.ok {
		return Some(r.value)
	}
	return None[T]()
}

// Chain provides a chainable API for Result operations.
type Chain[T, E any] struct {
	result Result[T, E]
}

// NewChain creates a new Chain from a Result.
func NewChain[T, E any](r Result[T, E]) *Chain[T, E] {
	return &Chain[T, E]{result: r}
}

// Map applies a transformation function.
func (c *Chain[T, E]) Map(fn func(T) T) *Chain[T, E] {
	c.result = Map(c.result, fn)
	return c
}

// FlatMap chains another Result-returning operation.
func (c *Chain[T, E]) FlatMap(fn func(T) Result[T, E]) *Chain[T, E] {
	c.result = FlatMap(c.result, fn)
	return c
}

// Filter applies a predicate.
func (c *Chain[T, E]) Filter(predicate func(T) bool, errorOnFalse E) *Chain[T, E] {
	c.result = Filter(c.result, predicate, errorOnFalse)
	return c
}

// Result returns the final Result.
func (c *Chain[T, E]) Result() Result[T, E] {
	return c.result
}

// Recovery provides panic recovery for Result operations.
func Recovery[T, E any](fn func() T, recoverFn func(any) E) Result[T, E] {
	defer func() {
		if r := recover(); r != nil {
			// This will be handled by the outer function
		}
	}()

	var result Result[T, E]
	func() {
		defer func() {
			if r := recover(); r != nil {
				result = Err[T, E](recoverFn(r))
			}
		}()
		result = Ok[T, E](fn())
	}()

	return result
}

// Try creates a Result from a function that may panic.
func Try[T any](fn func() T) Result[T, error] {
	return Recovery(fn, func(r any) error {
		return fmt.Errorf("panic recovered: %v", r)
	})
}

// Batch operations for multiple Results.

// All returns Ok if all Results are successful, otherwise returns the first error.
func All[T, E any](results ...Result[T, E]) Result[[]T, E] {
	values := make([]T, 0, len(results))
	for _, r := range results {
		if !r.ok {
			return Err[[]T, E](r.err)
		}
		values = append(values, r.value)
	}
	return Ok[[]T, E](values)
}

// Any returns the first successful Result, or the last error if all fail.
func Any[T, E any](results ...Result[T, E]) Result[T, E] {
	var lastErr E
	for _, r := range results {
		if r.ok {
			return r
		}
		lastErr = r.err
	}
	return Err[T, E](lastErr)
}

// Collect transforms a slice of values using a function that returns Results.
func Collect[T, U, E any](items []T, fn func(T) Result[U, E]) Result[[]U, E] {
	results := make([]Result[U, E], len(items))
	for i, item := range items {
		results[i] = fn(item)
	}
	return All(results...)
}

// Partition separates Results into successful values and errors.
func Partition[T, E any](results []Result[T, E]) ([]T, []E) {
	var values []T
	var errors []E

	for _, r := range results {
		if r.ok {
			values = append(values, r.value)
		} else {
			errors = append(errors, r.err)
		}
	}

	return values, errors
}
