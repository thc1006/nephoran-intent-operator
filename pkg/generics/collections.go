//go:build go1.24

package generics

import (
	"fmt"
	"maps"
	"slices"
	"sync"
)

// Comparable constraint for types that can be compared for equality.
type Comparable interface {
	comparable
}

// Ordered constraint for types that can be ordered.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

// Set provides a generic set implementation with O(1) operations.
// Thread-safe variant available via SafeSet.
type Set[T Comparable] struct {
	items map[T]struct{}
}

// NewSet creates a new generic Set with optional initial values.
func NewSet[T Comparable](values ...T) *Set[T] {
	s := &Set[T]{
		items: make(map[T]struct{}, len(values)),
	}
	for _, v := range values {
		s.items[v] = struct{}{}
	}
	return s
}

// Add inserts an item into the set.
func (s *Set[T]) Add(item T) {
	s.items[item] = struct{}{}
}

// Remove deletes an item from the set.
func (s *Set[T]) Remove(item T) {
	delete(s.items, item)
}

// Contains checks if an item exists in the set.
func (s *Set[T]) Contains(item T) bool {
	_, exists := s.items[item]
	return exists
}

// Size returns the number of items in the set.
func (s *Set[T]) Size() int {
	return len(s.items)
}

// IsEmpty returns true if the set is empty.
func (s *Set[T]) IsEmpty() bool {
	return len(s.items) == 0
}

// Clear removes all items from the set.
func (s *Set[T]) Clear() {
	clear(s.items)
}

// ToSlice returns all items as a slice.
func (s *Set[T]) ToSlice() []T {
	result := make([]T, 0, len(s.items))
	for item := range s.items {
		result = append(result, item)
	}
	return result
}

// Union returns a new set containing all items from both sets.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for item := range s.items {
		result.Add(item)
	}
	for item := range other.items {
		result.Add(item)
	}
	return result
}

// Intersection returns a new set containing items present in both sets.
func (s *Set[T]) Intersection(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for item := range s.items {
		if other.Contains(item) {
			result.Add(item)
		}
	}
	return result
}

// Difference returns a new set containing items in s but not in other.
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for item := range s.items {
		if !other.Contains(item) {
			result.Add(item)
		}
	}
	return result
}

// SymmetricDifference returns items in either set but not in both.
func (s *Set[T]) SymmetricDifference(other *Set[T]) *Set[T] {
	return s.Union(other).Difference(s.Intersection(other))
}

// IsSubset returns true if s is a subset of other.
func (s *Set[T]) IsSubset(other *Set[T]) bool {
	for item := range s.items {
		if !other.Contains(item) {
			return false
		}
	}
	return true
}

// IsSuperset returns true if s is a superset of other.
func (s *Set[T]) IsSuperset(other *Set[T]) bool {
	return other.IsSubset(s)
}

// ForEach applies a function to each item in the set.
func (s *Set[T]) ForEach(fn func(T)) {
	for item := range s.items {
		fn(item)
	}
}

// SafeSet provides a thread-safe version of Set.
type SafeSet[T Comparable] struct {
	set *Set[T]
	mu  sync.RWMutex
}

// NewSafeSet creates a new thread-safe Set.
func NewSafeSet[T Comparable](values ...T) *SafeSet[T] {
	return &SafeSet[T]{
		set: NewSet(values...),
	}
}

// Add thread-safely inserts an item.
func (s *SafeSet[T]) Add(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.set.Add(item)
}

// Remove thread-safely deletes an item.
func (s *SafeSet[T]) Remove(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.set.Remove(item)
}

// Contains thread-safely checks if an item exists.
func (s *SafeSet[T]) Contains(item T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.set.Contains(item)
}

// Size returns the number of items thread-safely.
func (s *SafeSet[T]) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.set.Size()
}

// ToSlice returns all items as a slice thread-safely.
func (s *SafeSet[T]) ToSlice() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.set.ToSlice()
}

// Map provides enhanced generic map operations.
type Map[K Comparable, V any] struct {
	items map[K]V
}

// NewMap creates a new generic Map.
func NewMap[K Comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		items: make(map[K]V),
	}
}

// FromMap creates a Map from a standard Go map.
func FromMap[K Comparable, V any](m map[K]V) *Map[K, V] {
	return &Map[K, V]{
		items: maps.Clone(m),
	}
}

// Set inserts or updates a key-value pair.
func (m *Map[K, V]) Set(key K, value V) {
	m.items[key] = value
}

// Get retrieves a value by key, returns Option.
func (m *Map[K, V]) Get(key K) Option[V] {
	if value, exists := m.items[key]; exists {
		return Some(value)
	}
	return None[V]()
}

// GetOrDefault retrieves a value by key with a default.
func (m *Map[K, V]) GetOrDefault(key K, defaultValue V) V {
	if value, exists := m.items[key]; exists {
		return value
	}
	return defaultValue
}

// Delete removes a key-value pair.
func (m *Map[K, V]) Delete(key K) {
	delete(m.items, key)
}

// Contains checks if a key exists.
func (m *Map[K, V]) Contains(key K) bool {
	_, exists := m.items[key]
	return exists
}

// Size returns the number of key-value pairs.
func (m *Map[K, V]) Size() int {
	return len(m.items)
}

// IsEmpty returns true if the map is empty.
func (m *Map[K, V]) IsEmpty() bool {
	return len(m.items) == 0
}

// Clear removes all key-value pairs.
func (m *Map[K, V]) Clear() {
	clear(m.items)
}

// Keys returns all keys as a slice.
func (m *Map[K, V]) Keys() []K {
	keys := make([]K, 0, len(m.items))
	for k := range m.items {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values as a slice.
func (m *Map[K, V]) Values() []V {
	values := make([]V, 0, len(m.items))
	for _, v := range m.items {
		values = append(values, v)
	}
	return values
}

// Entries returns all key-value pairs as a slice of tuples.
func (m *Map[K, V]) Entries() []Tuple[K, V] {
	entries := make([]Tuple[K, V], 0, len(m.items))
	for k, v := range m.items {
		entries = append(entries, NewTuple(k, v))
	}
	return entries
}

// ForEach applies a function to each key-value pair.
func (m *Map[K, V]) ForEach(fn func(K, V)) {
	for k, v := range m.items {
		fn(k, v)
	}
}

// MapValues transforms all values using a function.
func MapValues[K Comparable, V, U any](m *Map[K, V], fn func(V) U) *Map[K, U] {
	result := NewMap[K, U]()
	for k, v := range m.items {
		result.Set(k, fn(v))
	}
	return result
}

// FilterMap filters key-value pairs based on a predicate.
func FilterMap[K Comparable, V any](m *Map[K, V], predicate func(K, V) bool) *Map[K, V] {
	result := NewMap[K, V]()
	for k, v := range m.items {
		if predicate(k, v) {
			result.Set(k, v)
		}
	}
	return result
}

// ToMap returns the underlying map (copy).
func (m *Map[K, V]) ToMap() map[K]V {
	return maps.Clone(m.items)
}

// Slice provides enhanced slice operations with functional programming support.
type Slice[T any] struct {
	items []T
}

// NewSlice creates a new generic Slice.
func NewSlice[T any](items ...T) *Slice[T] {
	return &Slice[T]{
		items: slices.Clone(items),
	}
}

// FromSlice creates a Slice from a standard Go slice.
func FromSlice[T any](items []T) *Slice[T] {
	return &Slice[T]{
		items: slices.Clone(items),
	}
}

// Append adds items to the end of the slice.
func (s *Slice[T]) Append(items ...T) *Slice[T] {
	s.items = append(s.items, items...)
	return s
}

// Prepend adds items to the beginning of the slice.
func (s *Slice[T]) Prepend(items ...T) *Slice[T] {
	s.items = append(items, s.items...)
	return s
}

// Get retrieves an item by index, returns Option.
func (s *Slice[T]) Get(index int) Option[T] {
	if index >= 0 && index < len(s.items) {
		return Some(s.items[index])
	}
	return None[T]()
}

// Set updates an item by index if within bounds.
func (s *Slice[T]) Set(index int, value T) bool {
	if index >= 0 && index < len(s.items) {
		s.items[index] = value
		return true
	}
	return false
}

// Len returns the length of the slice.
func (s *Slice[T]) Len() int {
	return len(s.items)
}

// IsEmpty returns true if the slice is empty.
func (s *Slice[T]) IsEmpty() bool {
	return len(s.items) == 0
}

// First returns the first element as Option.
func (s *Slice[T]) First() Option[T] {
	if len(s.items) > 0 {
		return Some(s.items[0])
	}
	return None[T]()
}

// Last returns the last element as Option.
func (s *Slice[T]) Last() Option[T] {
	if len(s.items) > 0 {
		return Some(s.items[len(s.items)-1])
	}
	return None[T]()
}

// Map transforms each element using a function.
func MapSlice[T, U any](s *Slice[T], fn func(T) U) *Slice[U] {
	result := make([]U, len(s.items))
	for i, item := range s.items {
		result[i] = fn(item)
	}
	return FromSlice(result)
}

// Filter returns a new slice with elements that satisfy the predicate.
func FilterSlice[T any](s *Slice[T], predicate func(T) bool) *Slice[T] {
	var result []T
	for _, item := range s.items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return FromSlice(result)
}

// Reduce combines all elements using an accumulator function.
func ReduceSlice[T, U any](s *Slice[T], initialValue U, fn func(U, T) U) U {
	accumulator := initialValue
	for _, item := range s.items {
		accumulator = fn(accumulator, item)
	}
	return accumulator
}

// Find returns the first element that satisfies the predicate.
func FindSlice[T any](s *Slice[T], predicate func(T) bool) Option[T] {
	for _, item := range s.items {
		if predicate(item) {
			return Some(item)
		}
	}
	return None[T]()
}

// FindIndex returns the index of the first element that satisfies the predicate.
func FindIndexSlice[T any](s *Slice[T], predicate func(T) bool) Option[int] {
	for i, item := range s.items {
		if predicate(item) {
			return Some(i)
		}
	}
	return None[int]()
}

// Any returns true if any element satisfies the predicate.
func AnySlice[T any](s *Slice[T], predicate func(T) bool) bool {
	for _, item := range s.items {
		if predicate(item) {
			return true
		}
	}
	return false
}

// All returns true if all elements satisfy the predicate.
func AllSlice[T any](s *Slice[T], predicate func(T) bool) bool {
	for _, item := range s.items {
		if !predicate(item) {
			return false
		}
	}
	return true
}

// Contains checks if the slice contains the given item (requires comparable).
func ContainsSlice[T Comparable](s *Slice[T], item T) bool {
	return slices.Contains(s.items, item)
}

// Sort sorts the slice in place (requires ordered).
func SortSlice[T Ordered](s *Slice[T]) *Slice[T] {
	slices.Sort(s.items)
	return s
}

// SortBy sorts the slice using a custom comparison function.
func SortBySlice[T any](s *Slice[T], cmp func(T, T) int) *Slice[T] {
	slices.SortFunc(s.items, cmp)
	return s
}

// Reverse reverses the slice in place.
func ReverseSlice[T any](s *Slice[T]) *Slice[T] {
	slices.Reverse(s.items)
	return s
}

// Unique returns a new slice with unique elements (requires comparable).
func UniqueSlice[T Comparable](s *Slice[T]) *Slice[T] {
	seen := NewSet[T]()
	var result []T
	for _, item := range s.items {
		if !seen.Contains(item) {
			seen.Add(item)
			result = append(result, item)
		}
	}
	return FromSlice(result)
}

// Chunk splits the slice into chunks of specified size.
func ChunkSlice[T any](s *Slice[T], size int) []*Slice[T] {
	if size <= 0 {
		return nil
	}

	var chunks []*Slice[T]
	for i := 0; i < len(s.items); i += size {
		end := i + size
		if end > len(s.items) {
			end = len(s.items)
		}
		chunks = append(chunks, FromSlice(s.items[i:end]))
	}
	return chunks
}

// ToSlice returns the underlying slice (copy).
func (s *Slice[T]) ToSlice() []T {
	return slices.Clone(s.items)
}

// ForEach applies a function to each element.
func (s *Slice[T]) ForEach(fn func(T)) {
	for _, item := range s.items {
		fn(item)
	}
}

// ForEachIndex applies a function to each element with its index.
func (s *Slice[T]) ForEachIndex(fn func(int, T)) {
	for i, item := range s.items {
		fn(i, item)
	}
}

// Tuple represents a pair of values with type safety.
type Tuple[T, U any] struct {
	First  T
	Second U
}

// NewTuple creates a new Tuple.
func NewTuple[T, U any](first T, second U) Tuple[T, U] {
	return Tuple[T, U]{
		First:  first,
		Second: second,
	}
}

// Unpack returns the tuple values.
func (t Tuple[T, U]) Unpack() (T, U) {
	return t.First, t.Second
}

// String provides string representation.
func (t Tuple[T, U]) String() string {
	return fmt.Sprintf("(%v, %v)", t.First, t.Second)
}

// Triple represents three values.
type Triple[T, U, V any] struct {
	First  T
	Second U
	Third  V
}

// NewTriple creates a new Triple.
func NewTriple[T, U, V any](first T, second U, third V) Triple[T, U, V] {
	return Triple[T, U, V]{
		First:  first,
		Second: second,
		Third:  third,
	}
}

// Unpack returns the triple values.
func (t Triple[T, U, V]) Unpack() (T, U, V) {
	return t.First, t.Second, t.Third
}

// String provides string representation.
func (t Triple[T, U, V]) String() string {
	return fmt.Sprintf("(%v, %v, %v)", t.First, t.Second, t.Third)
}

// Utility functions for batch operations

// ZipSlices combines two slices into tuples.
func ZipSlices[T, U any](slice1 *Slice[T], slice2 *Slice[U]) *Slice[Tuple[T, U]] {
	minLen := min(slice1.Len(), slice2.Len())
	result := make([]Tuple[T, U], minLen)

	for i := 0; i < minLen; i++ {
		result[i] = NewTuple(slice1.items[i], slice2.items[i])
	}

	return FromSlice(result)
}

// FlattenSlices flattens a slice of slices.
func FlattenSlices[T any](slices []*Slice[T]) *Slice[T] {
	var totalLen int
	for _, s := range slices {
		totalLen += s.Len()
	}

	result := make([]T, 0, totalLen)
	for _, s := range slices {
		result = append(result, s.items...)
	}

	return FromSlice(result)
}
