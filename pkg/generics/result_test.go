//go:build go1.24

package generics

import (
	"fmt"
	"testing"
)

func TestResult_Ok(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"string value", "hello"},
		{"empty string", ""},
		{"number as string", "42"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Ok[string, error](tt.value)
			
			if !result.IsOk() {
				t.Errorf("Expected Ok result, got error")
			}
			
			if result.IsErr() {
				t.Errorf("Expected Ok result, IsErr() returned true")
			}
			
			if result.Value() != tt.value {
				t.Errorf("Expected value %s, got %s", tt.value, result.Value())
			}
		})
	}
}

func TestResult_Err(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"simple error", fmt.Errorf("test error")},
		{"nil error", nil},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Err[string, error](tt.err)
			
			if result.IsOk() {
				t.Errorf("Expected error result, got Ok")
			}
			
			if !result.IsErr() {
				t.Errorf("Expected error result, IsErr() returned false")
			}
			
			if result.Error() != tt.err {
				t.Errorf("Expected error %v, got %v", tt.err, result.Error())
			}
		})
	}
}

func TestResult_Map(t *testing.T) {
	tests := []struct {
		name     string
		input    Result[int, error]
		fn       func(int) string
		expected string
		isOk     bool
	}{
		{
			name:     "map successful result",
			input:    Ok[int, error](42),
			fn:       func(i int) string { return fmt.Sprintf("value: %d", i) },
			expected: "value: 42",
			isOk:     true,
		},
		{
			name:     "map error result",
			input:    Err[int, error](fmt.Errorf("test error")),
			fn:       func(i int) string { return fmt.Sprintf("value: %d", i) },
			expected: "",
			isOk:     false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Map(tt.input, tt.fn)
			
			if result.IsOk() != tt.isOk {
				t.Errorf("Expected isOk %v, got %v", tt.isOk, result.IsOk())
			}
			
			if tt.isOk && result.Value() != tt.expected {
				t.Errorf("Expected value %s, got %s", tt.expected, result.Value())
			}
		})
	}
}

func TestResult_FlatMap(t *testing.T) {
	tests := []struct {
		name     string
		input    Result[int, error]
		fn       func(int) Result[string, error]
		expected string
		isOk     bool
	}{
		{
			name:  "flatmap successful result",
			input: Ok[int, error](42),
			fn: func(i int) Result[string, error] {
				return Ok[string, error](fmt.Sprintf("value: %d", i))
			},
			expected: "value: 42",
			isOk:     true,
		},
		{
			name:  "flatmap with function returning error",
			input: Ok[int, error](42),
			fn: func(i int) Result[string, error] {
				return Err[string, error](fmt.Errorf("function error"))
			},
			expected: "",
			isOk:     false,
		},
		{
			name:  "flatmap error input",
			input: Err[int, error](fmt.Errorf("input error")),
			fn: func(i int) Result[string, error] {
				return Ok[string, error](fmt.Sprintf("value: %d", i))
			},
			expected: "",
			isOk:     false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FlatMap(tt.input, tt.fn)
			
			if result.IsOk() != tt.isOk {
				t.Errorf("Expected isOk %v, got %v", tt.isOk, result.IsOk())
			}
			
			if tt.isOk && result.Value() != tt.expected {
				t.Errorf("Expected value %s, got %s", tt.expected, result.Value())
			}
		})
	}
}

func TestResult_Filter(t *testing.T) {
	tests := []struct {
		name      string
		input     Result[int, error]
		predicate func(int) bool
		errorOnFalse error
		isOk      bool
	}{
		{
			name:         "filter passing predicate",
			input:        Ok[int, error](42),
			predicate:    func(i int) bool { return i > 30 },
			errorOnFalse: fmt.Errorf("value too small"),
			isOk:         true,
		},
		{
			name:         "filter failing predicate",
			input:        Ok[int, error>(10),
			predicate:    func(i int) bool { return i > 30 },
			errorOnFalse: fmt.Errorf("value too small"),
			isOk:         false,
		},
		{
			name:         "filter error input",
			input:        Err[int, error](fmt.Errorf("input error")),
			predicate:    func(i int) bool { return i > 30 },
			errorOnFalse: fmt.Errorf("value too small"),
			isOk:         false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Filter(tt.input, tt.predicate, tt.errorOnFalse)
			
			if result.IsOk() != tt.isOk {
				t.Errorf("Expected isOk %v, got %v", tt.isOk, result.IsOk())
			}
		})
	}
}

func TestOption_Some(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"positive number", 42},
		{"zero", 0},
		{"negative number", -1},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := Some(tt.value)
			
			if !option.IsSome() {
				t.Errorf("Expected Some option")
			}
			
			if option.IsNone() {
				t.Errorf("Expected Some option, IsNone() returned true")
			}
			
			if option.Value() != tt.value {
				t.Errorf("Expected value %d, got %d", tt.value, option.Value())
			}
		})
	}
}

func TestOption_None(t *testing.T) {
	option := None[string]()
	
	if option.IsSome() {
		t.Errorf("Expected None option, IsSome() returned true")
	}
	
	if !option.IsNone() {
		t.Errorf("Expected None option")
	}
}

func TestOption_ValueOr(t *testing.T) {
	tests := []struct {
		name         string
		option       Option[string]
		defaultValue string
		expected     string
	}{
		{
			name:         "some option",
			option:       Some("hello"),
			defaultValue: "default",
			expected:     "hello",
		},
		{
			name:         "none option",
			option:       None[string](),
			defaultValue: "default",
			expected:     "default",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.option.ValueOr(tt.defaultValue)
			
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMapOption(t *testing.T) {
	tests := []struct {
		name     string
		input    Option[int]
		fn       func(int) string
		expected Option[string]
	}{
		{
			name:     "map some option",
			input:    Some(42),
			fn:       func(i int) string { return fmt.Sprintf("value: %d", i) },
			expected: Some("value: 42"),
		},
		{
			name:     "map none option",
			input:    None[int](),
			fn:       func(i int) string { return fmt.Sprintf("value: %d", i) },
			expected: None[string](),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapOption(tt.input, tt.fn)
			
			if result.IsSome() != tt.expected.IsSome() {
				t.Errorf("Expected IsSome %v, got %v", tt.expected.IsSome(), result.IsSome())
			}
			
			if result.IsSome() {
				if result.Value() != tt.expected.Value() {
					t.Errorf("Expected value %s, got %s", tt.expected.Value(), result.Value())
				}
			}
		})
	}
}

func TestAll(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result[int, error]
		expected []int
		isOk     bool
	}{
		{
			name: "all successful",
			results: []Result[int, error]{
				Ok[int, error](1),
				Ok[int, error](2),
				Ok[int, error](3),
			},
			expected: []int{1, 2, 3},
			isOk:     true,
		},
		{
			name: "one error",
			results: []Result[int, error]{
				Ok[int, error](1),
				Err[int, error](fmt.Errorf("test error")),
				Ok[int, error](3),
			},
			expected: nil,
			isOk:     false,
		},
		{
			name:     "empty slice",
			results:  []Result[int, error]{},
			expected: []int{},
			isOk:     true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := All(tt.results...)
			
			if result.IsOk() != tt.isOk {
				t.Errorf("Expected isOk %v, got %v", tt.isOk, result.IsOk())
			}
			
			if tt.isOk {
				values := result.Value()
				if len(values) != len(tt.expected) {
					t.Errorf("Expected %d values, got %d", len(tt.expected), len(values))
				}
				
				for i, expected := range tt.expected {
					if values[i] != expected {
						t.Errorf("Expected value[%d] = %d, got %d", i, expected, values[i])
					}
				}
			}
		})
	}
}

func TestAny(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result[int, error]
		expected int
		isOk     bool
	}{
		{
			name: "first successful",
			results: []Result[int, error>{
				Ok[int, error](1),
				Err[int, error](fmt.Errorf("test error")),
				Ok[int, error](3),
			},
			expected: 1,
			isOk:     true,
		},
		{
			name: "all errors",
			results: []Result[int, error]{
				Err[int, error](fmt.Errorf("error 1")),
				Err[int, error](fmt.Errorf("error 2")),
				Err[int, error](fmt.Errorf("error 3")),
			},
			expected: 0,
			isOk:     false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Any(tt.results...)
			
			if result.IsOk() != tt.isOk {
				t.Errorf("Expected isOk %v, got %v", tt.isOk, result.IsOk())
			}
			
			if tt.isOk && result.Value() != tt.expected {
				t.Errorf("Expected value %d, got %d", tt.expected, result.Value())
			}
		})
	}
}

func TestChain(t *testing.T) {
	multiply := func(x int) int { return x * 2 }
	add := func(x int) int { return x + 10 }
	
	tests := []struct {
		name     string
		initial  Result[int, error]
		expected int
		isOk     bool
	}{
		{
			name:     "successful chain",
			initial:  Ok[int, error](5),
			expected: 20, // (5 * 2) + 10
			isOk:     true,
		},
		{
			name:     "error in chain",
			initial:  Err[int, error](fmt.Errorf("initial error")),
			expected: 0,
			isOk:     false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewChain(tt.initial).
				Map(multiply).
				Map(add).
				Result()
			
			if result.IsOk() != tt.isOk {
				t.Errorf("Expected isOk %v, got %v", tt.isOk, result.IsOk())
			}
			
			if tt.isOk && result.Value() != tt.expected {
				t.Errorf("Expected value %d, got %d", tt.expected, result.Value())
			}
		})
	}
}

func TestTry(t *testing.T) {
	tests := []struct {
		name string
		fn   func() int
		isOk bool
	}{
		{
			name: "successful function",
			fn:   func() int { return 42 },
			isOk: true,
		},
		{
			name: "panicking function",
			fn:   func() int { panic("test panic") },
			isOk: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Try(tt.fn)
			
			if result.IsOk() != tt.isOk {
				t.Errorf("Expected isOk %v, got %v", tt.isOk, result.IsOk())
			}
		})
	}
}

func TestPartition(t *testing.T) {
	results := []Result[int, string>{
		Ok[int, string](1),
		Err[int, string]("error1"),
		Ok[int, string](2),
		Err[int, string]("error2"),
		Ok[int, string](3),
	}
	
	values, errors := Partition(results)
	
	expectedValues := []int{1, 2, 3}
	expectedErrors := []string{"error1", "error2"}
	
	if len(values) != len(expectedValues) {
		t.Errorf("Expected %d values, got %d", len(expectedValues), len(values))
	}
	
	for i, expected := range expectedValues {
		if values[i] != expected {
			t.Errorf("Expected value[%d] = %d, got %d", i, expected, values[i])
		}
	}
	
	if len(errors) != len(expectedErrors) {
		t.Errorf("Expected %d errors, got %d", len(expectedErrors), len(errors))
	}
	
	for i, expected := range expectedErrors {
		if errors[i] != expected {
			t.Errorf("Expected error[%d] = %s, got %s", i, expected, errors[i])
		}
	}
}

// Benchmarks

func BenchmarkResult_Ok(b *testing.B) {
	for i := 0; i < b.N; i++ {
		result := Ok[int, error](42)
		_ = result.Value()
	}
}

func BenchmarkResult_Map(b *testing.B) {
	result := Ok[int, error](42)
	fn := func(x int) int { return x * 2 }
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Map(result, fn)
	}
}

func BenchmarkOption_Some(b *testing.B) {
	for i := 0; i < b.N; i++ {
		option := Some(42)
		_ = option.Value()
	}
}

func BenchmarkChain(b *testing.B) {
	initial := Ok[int, error](5)
	multiply := func(x int) int { return x * 2 }
	add := func(x int) int { return x + 10 }
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewChain(initial).Map(multiply).Map(add).Result()
	}
}