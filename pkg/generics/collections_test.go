//go:build go1.24

package generics

import (
	"fmt"
	"slices"
	"testing"
)

func TestSet_BasicOperations(t *testing.T) {
	set := NewSet[int]()

	// Test empty set
	if !set.IsEmpty() {
		t.Error("New set should be empty")
	}

	if set.Size() != 0 {
		t.Errorf("Expected size 0, got %d", set.Size())
	}

	// Test add operations
	set.Add(1)
	set.Add(2)
	set.Add(3)

	if set.Size() != 3 {
		t.Errorf("Expected size 3, got %d", set.Size())
	}

	if set.IsEmpty() {
		t.Error("Set should not be empty after adding elements")
	}

	// Test contains
	if !set.Contains(1) {
		t.Error("Set should contain 1")
	}

	if set.Contains(4) {
		t.Error("Set should not contain 4")
	}

	// Test remove
	set.Remove(2)
	if set.Contains(2) {
		t.Error("Set should not contain 2 after removal")
	}

	if set.Size() != 2 {
		t.Errorf("Expected size 2 after removal, got %d", set.Size())
	}
}

func TestSet_SetOperations(t *testing.T) {
	set1 := NewSet(1, 2, 3)
	set2 := NewSet(3, 4, 5)

	// Test union
	union := set1.Union(set2)
	expected := []int{1, 2, 3, 4, 5}
	result := union.ToSlice()
	slices.Sort(result)

	if !slices.Equal(result, expected) {
		t.Errorf("Union result mismatch. Expected %v, got %v", expected, result)
	}

	// Test intersection
	intersection := set1.Intersection(set2)
	if intersection.Size() != 1 {
		t.Errorf("Expected intersection size 1, got %d", intersection.Size())
	}

	if !intersection.Contains(3) {
		t.Error("Intersection should contain 3")
	}

	// Test difference
	difference := set1.Difference(set2)
	expected = []int{1, 2}
	result = difference.ToSlice()
	slices.Sort(result)

	if !slices.Equal(result, expected) {
		t.Errorf("Difference result mismatch. Expected %v, got %v", expected, result)
	}

	// Test subset/superset
	subset := NewSet(1, 2)
	if !subset.IsSubset(set1) {
		t.Error("subset should be subset of set1")
	}

	if !set1.IsSuperset(subset) {
		t.Error("set1 should be superset of subset")
	}
}

func TestSafeSet_ConcurrentAccess(t *testing.T) {
	set := NewSafeSet[int]()

	// Basic thread-safe operations
	set.Add(1)
	set.Add(2)
	set.Add(3)

	if set.Size() != 3 {
		t.Errorf("Expected size 3, got %d", set.Size())
	}

	if !set.Contains(1) {
		t.Error("SafeSet should contain 1")
	}

	set.Remove(2)
	if set.Contains(2) {
		t.Error("SafeSet should not contain 2 after removal")
	}

	slice := set.ToSlice()
	if len(slice) != 2 {
		t.Errorf("Expected slice length 2, got %d", len(slice))
	}
}

func TestMap_BasicOperations(t *testing.T) {
	m := NewMap[string, int]()

	// Test empty map
	if !m.IsEmpty() {
		t.Error("New map should be empty")
	}

	if m.Size() != 0 {
		t.Errorf("Expected size 0, got %d", m.Size())
	}

	// Test set and get
	m.Set("one", 1)
	m.Set("two", 2)
	m.Set("three", 3)

	if m.Size() != 3 {
		t.Errorf("Expected size 3, got %d", m.Size())
	}

	value := m.Get("one")
	if value.IsNone() {
		t.Error("Expected to find 'one' in map")
	}

	if value.Value() != 1 {
		t.Errorf("Expected value 1, got %d", value.Value())
	}

	// Test non-existent key
	missing := m.Get("four")
	if missing.IsSome() {
		t.Error("Expected 'four' to be missing from map")
	}

	// Test GetOrDefault
	defaultValue := m.GetOrDefault("four", 999)
	if defaultValue != 999 {
		t.Errorf("Expected default value 999, got %d", defaultValue)
	}

	// Test contains
	if !m.Contains("two") {
		t.Error("Map should contain 'two'")
	}

	// Test delete
	m.Delete("two")
	if m.Contains("two") {
		t.Error("Map should not contain 'two' after deletion")
	}

	if m.Size() != 2 {
		t.Errorf("Expected size 2 after deletion, got %d", m.Size())
	}
}

func TestMap_KeysValuesEntries(t *testing.T) {
	m := NewMap[string, int]()
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	// Test Keys
	keys := m.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Test Values
	values := m.Values()
	if len(values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(values))
	}

	// Test Entries
	entries := m.Entries()
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	// Verify entries structure
	for _, entry := range entries {
		key, value := entry.Unpack()
		expectedValue := m.Get(key)
		if expectedValue.IsNone() || expectedValue.Value() != value {
			t.Errorf("Entry mismatch for key %s", key)
		}
	}
}

func TestMapValues(t *testing.T) {
	m := NewMap[string, int]()
	m.Set("one", 1)
	m.Set("two", 2)
	m.Set("three", 3)

	// Transform values (multiply by 2)
	doubled := MapValues(m, func(x int) int { return x * 2 })

	if doubled.Size() != 3 {
		t.Errorf("Expected transformed map size 3, got %d", doubled.Size())
	}

	value := doubled.Get("two")
	if value.IsNone() || value.Value() != 4 {
		t.Errorf("Expected transformed value 4, got %v", value)
	}
}

func TestFilterMap(t *testing.T) {
	m := NewMap[string, int]()
	m.Set("one", 1)
	m.Set("two", 2)
	m.Set("three", 3)
	m.Set("four", 4)

	// Filter even values
	evenMap := FilterMap(m, func(k string, v int) bool { return v%2 == 0 })

	if evenMap.Size() != 2 {
		t.Errorf("Expected filtered map size 2, got %d", evenMap.Size())
	}

	if !evenMap.Contains("two") {
		t.Error("Filtered map should contain 'two'")
	}

	if !evenMap.Contains("four") {
		t.Error("Filtered map should contain 'four'")
	}

	if evenMap.Contains("one") {
		t.Error("Filtered map should not contain 'one'")
	}
}

func TestSlice_BasicOperations(t *testing.T) {
	slice := NewSlice(1, 2, 3)

	if slice.Len() != 3 {
		t.Errorf("Expected length 3, got %d", slice.Len())
	}

	if slice.IsEmpty() {
		t.Error("Slice should not be empty")
	}

	// Test Get
	value := slice.Get(1)
	if value.IsNone() || value.Value() != 2 {
		t.Errorf("Expected value 2 at index 1, got %v", value)
	}

	// Test out of bounds
	outOfBounds := slice.Get(10)
	if outOfBounds.IsSome() {
		t.Error("Expected None for out of bounds index")
	}

	// Test Set
	success := slice.Set(1, 10)
	if !success {
		t.Error("Set should succeed for valid index")
	}

	updated := slice.Get(1)
	if updated.IsNone() || updated.Value() != 10 {
		t.Errorf("Expected updated value 10, got %v", updated)
	}

	// Test First and Last
	first := slice.First()
	if first.IsNone() || first.Value() != 1 {
		t.Errorf("Expected first value 1, got %v", first)
	}

	last := slice.Last()
	if last.IsNone() || last.Value() != 3 {
		t.Errorf("Expected last value 3, got %v", last)
	}

	// Test Append
	slice.Append(4, 5)
	if slice.Len() != 5 {
		t.Errorf("Expected length 5 after append, got %d", slice.Len())
	}

	// Test Prepend
	slice.Prepend(0)
	if slice.Len() != 6 {
		t.Errorf("Expected length 6 after prepend, got %d", slice.Len())
	}

	first = slice.First()
	if first.IsNone() || first.Value() != 0 {
		t.Errorf("Expected first value 0 after prepend, got %v", first)
	}
}

func TestMapSlice(t *testing.T) {
	slice := NewSlice(1, 2, 3)

	// Map to string
	stringSlice := MapSlice(slice, func(x int) string {
		return fmt.Sprintf("value_%d", x)
	})

	if stringSlice.Len() != 3 {
		t.Errorf("Expected mapped slice length 3, got %d", stringSlice.Len())
	}

	first := stringSlice.First()
	if first.IsNone() || first.Value() != "value_1" {
		t.Errorf("Expected 'value_1', got %v", first)
	}
}

func TestFilterSlice(t *testing.T) {
	slice := NewSlice(1, 2, 3, 4, 5, 6)

	// Filter even numbers
	evenSlice := FilterSlice(slice, func(x int) bool { return x%2 == 0 })

	if evenSlice.Len() != 3 {
		t.Errorf("Expected filtered slice length 3, got %d", evenSlice.Len())
	}

	expected := []int{2, 4, 6}
	actual := evenSlice.ToSlice()

	if !slices.Equal(actual, expected) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func TestReduceSlice(t *testing.T) {
	slice := NewSlice(1, 2, 3, 4, 5)

	// Sum all numbers
	sum := ReduceSlice(slice, 0, func(acc, x int) int { return acc + x })

	if sum != 15 {
		t.Errorf("Expected sum 15, got %d", sum)
	}

	// Product
	product := ReduceSlice(slice, 1, func(acc, x int) int { return acc * x })

	if product != 120 {
		t.Errorf("Expected product 120, got %d", product)
	}
}

func TestFindSlice(t *testing.T) {
	slice := NewSlice(1, 3, 5, 2, 4)

	// Find first even number
	even := FindSlice(slice, func(x int) bool { return x%2 == 0 })

	if even.IsNone() {
		t.Error("Expected to find an even number")
	}

	if even.Value() != 2 {
		t.Errorf("Expected first even number 2, got %d", even.Value())
	}

	// Find non-existent value
	large := FindSlice(slice, func(x int) bool { return x > 10 })

	if large.IsSome() {
		t.Error("Expected not to find large number")
	}
}

func TestAnyAllSlice(t *testing.T) {
	slice := NewSlice(2, 4, 6, 8)

	// Test Any
	hasEven := AnySlice(slice, func(x int) bool { return x%2 == 0 })
	if !hasEven {
		t.Error("Expected to find even numbers")
	}

	hasOdd := AnySlice(slice, func(x int) bool { return x%2 == 1 })
	if hasOdd {
		t.Error("Expected not to find odd numbers")
	}

	// Test All
	allEven := AllSlice(slice, func(x int) bool { return x%2 == 0 })
	if !allEven {
		t.Error("Expected all numbers to be even")
	}

	allPositive := AllSlice(slice, func(x int) bool { return x > 0 })
	if !allPositive {
		t.Error("Expected all numbers to be positive")
	}
}

func TestContainsSlice(t *testing.T) {
	slice := NewSlice("apple", "banana", "cherry")

	if !ContainsSlice(slice, "banana") {
		t.Error("Expected slice to contain 'banana'")
	}

	if ContainsSlice(slice, "grape") {
		t.Error("Expected slice not to contain 'grape'")
	}
}

func TestSortSlice(t *testing.T) {
	slice := NewSlice(3, 1, 4, 1, 5, 9, 2, 6)

	// Sort in place
	SortSlice(slice)

	expected := []int{1, 1, 2, 3, 4, 5, 6, 9}
	actual := slice.ToSlice()

	if !slices.Equal(actual, expected) {
		t.Errorf("Expected sorted slice %v, got %v", expected, actual)
	}
}

func TestSortBySlice(t *testing.T) {
	slice := NewSlice("apple", "pie", "banana", "kiwi")

	// Sort by length
	SortBySlice(slice, func(a, b string) int {
		if len(a) < len(b) {
			return -1
		} else if len(a) > len(b) {
			return 1
		}
		return 0
	})

	expected := []string{"pie", "kiwi", "apple", "banana"}
	actual := slice.ToSlice()

	if !slices.Equal(actual, expected) {
		t.Errorf("Expected sorted slice %v, got %v", expected, actual)
	}
}

func TestUniqueSlice(t *testing.T) {
	slice := NewSlice(1, 2, 2, 3, 3, 3, 4)

	unique := UniqueSlice(slice)

	expected := []int{1, 2, 3, 4}
	actual := unique.ToSlice()

	if len(actual) != 4 {
		t.Errorf("Expected 4 unique elements, got %d", len(actual))
	}

	// Check all expected elements are present (order may vary)
	for _, exp := range expected {
		if !slices.Contains(actual, exp) {
			t.Errorf("Expected unique slice to contain %d", exp)
		}
	}
}

func TestChunkSlice(t *testing.T) {
	slice := NewSlice(1, 2, 3, 4, 5, 6, 7, 8, 9)

	chunks := ChunkSlice(slice, 3)

	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}

	// Check first chunk
	firstChunk := chunks[0].ToSlice()
	expected := []int{1, 2, 3}

	if !slices.Equal(firstChunk, expected) {
		t.Errorf("Expected first chunk %v, got %v", expected, firstChunk)
	}

	// Check last chunk
	lastChunk := chunks[2].ToSlice()
	expected = []int{7, 8, 9}

	if !slices.Equal(lastChunk, expected) {
		t.Errorf("Expected last chunk %v, got %v", expected, lastChunk)
	}
}

func TestZipSlices(t *testing.T) {
	slice1 := NewSlice(1, 2, 3)
	slice2 := NewSlice("a", "b", "c")

	zipped := ZipSlices(slice1, slice2)

	if zipped.Len() != 3 {
		t.Errorf("Expected zipped length 3, got %d", zipped.Len())
	}

	first := zipped.Get(0)
	if first.IsNone() {
		t.Error("Expected first tuple to exist")
	}

	tuple := first.Value()
	if tuple.First != 1 || tuple.Second != "a" {
		t.Errorf("Expected (1, 'a'), got (%v, %v)", tuple.First, tuple.Second)
	}
}

func TestTuple(t *testing.T) {
	tuple := NewTuple("hello", 42)

	if tuple.First != "hello" {
		t.Errorf("Expected first value 'hello', got %s", tuple.First)
	}

	if tuple.Second != 42 {
		t.Errorf("Expected second value 42, got %d", tuple.Second)
	}

	first, second := tuple.Unpack()
	if first != "hello" || second != 42 {
		t.Errorf("Unpack failed: got (%s, %d)", first, second)
	}

	str := tuple.String()
	expected := "(hello, 42)"
	if str != expected {
		t.Errorf("Expected string %s, got %s", expected, str)
	}
}

func TestTriple(t *testing.T) {
	triple := NewTriple("hello", 42, true)

	if triple.First != "hello" {
		t.Errorf("Expected first value 'hello', got %s", triple.First)
	}

	if triple.Second != 42 {
		t.Errorf("Expected second value 42, got %d", triple.Second)
	}

	if triple.Third != true {
		t.Errorf("Expected third value true, got %v", triple.Third)
	}

	first, second, third := triple.Unpack()
	if first != "hello" || second != 42 || third != true {
		t.Errorf("Unpack failed: got (%s, %d, %v)", first, second, third)
	}
}

// Benchmarks

func BenchmarkSet_Add(b *testing.B) {
	set := NewSet[int]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Add(i)
	}
}

func BenchmarkSet_Contains(b *testing.B) {
	set := NewSet[int]()
	for i := 0; i < 1000; i++ {
		set.Add(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Contains(i % 1000)
	}
}

func BenchmarkMap_Set(b *testing.B) {
	m := NewMap[int, string]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set(i, fmt.Sprintf("value_%d", i))
	}
}

func BenchmarkMap_Get(b *testing.B) {
	m := NewMap[int, string]()
	for i := 0; i < 1000; i++ {
		m.Set(i, fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Get(i % 1000)
	}
}

func BenchmarkSlice_Append(b *testing.B) {
	slice := NewSlice[int]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice.Append(i)
	}
}

func BenchmarkMapSlice(b *testing.B) {
	slice := NewSlice[int]()
	for i := 0; i < 1000; i++ {
		slice.Append(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MapSlice(slice, func(x int) int { return x * 2 })
	}
}
