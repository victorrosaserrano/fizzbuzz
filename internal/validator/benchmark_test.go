package validator

import (
	"regexp"
	"testing"
)

// BenchmarkValidator tests performance of core validator operations
func BenchmarkValidator(b *testing.B) {
	b.Run("New", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = New()
		}
	})

	b.Run("AddError", func(b *testing.B) {
		v := New()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			v.AddError("field", "error message")
		}
	})

	b.Run("Check-True", func(b *testing.B) {
		v := New()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			v.Check(true, "field", "error message")
		}
	})

	b.Run("Check-False", func(b *testing.B) {
		v := New()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			v.Check(false, "field", "error message")
		}
	})

	b.Run("Valid", func(b *testing.B) {
		v := New()
		v.AddError("field", "error")
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = v.Valid()
		}
	})

	b.Run("ErrorMap", func(b *testing.B) {
		v := New()
		v.AddError("field1", "error1")
		v.AddError("field2", "error2")
		v.AddError("field3", "error3")
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = v.ErrorMap()
		}
	})

	b.Run("Clear", func(b *testing.B) {
		v := New()
		for i := 0; i < 10; i++ {
			v.AddError("field", "error")
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			v.Clear()
		}
	})
}

// BenchmarkValidationHelpers tests performance of validation helper functions
func BenchmarkValidationHelpers(b *testing.B) {
	b.Run("PermittedValue-String", func(b *testing.B) {
		permitted := []string{"apple", "banana", "cherry", "date", "elderberry"}

		for i := 0; i < b.N; i++ {
			_ = PermittedValue("cherry", permitted...)
		}
	})

	b.Run("PermittedValue-Int", func(b *testing.B) {
		permitted := []int{1, 3, 5, 7, 9, 11, 13, 15, 17, 19}

		for i := 0; i < b.N; i++ {
			_ = PermittedValue(7, permitted...)
		}
	})

	b.Run("In-String", func(b *testing.B) {
		list := []string{"apple", "banana", "cherry", "date", "elderberry"}

		for i := 0; i < b.N; i++ {
			_ = In("cherry", list)
		}
	})

	b.Run("In-Int", func(b *testing.B) {
		list := []int{1, 3, 5, 7, 9, 11, 13, 15, 17, 19}

		for i := 0; i < b.N; i++ {
			_ = In(7, list)
		}
	})

	b.Run("Unique-String-Unique", func(b *testing.B) {
		values := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}

		for i := 0; i < b.N; i++ {
			_ = Unique(values)
		}
	})

	b.Run("Unique-String-Duplicate", func(b *testing.B) {
		values := []string{"a", "b", "c", "d", "a", "f", "g", "h", "i", "j"}

		for i := 0; i < b.N; i++ {
			_ = Unique(values)
		}
	})

	b.Run("Unique-Int-Unique", func(b *testing.B) {
		values := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

		for i := 0; i < b.N; i++ {
			_ = Unique(values)
		}
	})

	b.Run("Matches-Email", func(b *testing.B) {
		pattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		email := "test@example.com"

		for i := 0; i < b.N; i++ {
			_ = Matches(email, pattern)
		}
	})

	b.Run("Matches-Simple", func(b *testing.B) {
		pattern := regexp.MustCompile(`^\d+$`)
		number := "12345"

		for i := 0; i < b.N; i++ {
			_ = Matches(number, pattern)
		}
	})
}

// BenchmarkFizzBuzzValidation simulates real-world FizzBuzz parameter validation
func BenchmarkFizzBuzzValidation(b *testing.B) {
	b.Run("Valid-Parameters", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v := New()

			// Typical FizzBuzz validation
			int1, int2, limit := 3, 5, 100
			str1, str2 := "fizz", "buzz"

			v.Check(int1 > 0, "int1", "must be a positive integer")
			v.Check(int1 <= 10000, "int1", "must not be more than 10,000")
			v.Check(int2 > 0, "int2", "must be a positive integer")
			v.Check(int2 <= 10000, "int2", "must not be more than 10,000")
			v.Check(int1 != int2, "int1", "must be different from int2")
			v.Check(limit > 0, "limit", "must be a positive integer")
			v.Check(limit <= 100000, "limit", "must not be more than 100,000")
			v.Check(str1 != "", "str1", "must be provided")
			v.Check(len(str1) <= 50, "str1", "must not be more than 50 characters")
			v.Check(str2 != "", "str2", "must be provided")
			v.Check(len(str2) <= 50, "str2", "must not be more than 50 characters")

			_ = v.Valid()
		}
	})

	b.Run("Invalid-Parameters", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v := New()

			// Invalid FizzBuzz parameters that will generate errors
			int1, int2, limit := -1, -1, 0
			str1, str2 := "", string(make([]byte, 60))

			v.Check(int1 > 0, "int1", "must be a positive integer")
			v.Check(int1 <= 10000, "int1", "must not be more than 10,000")
			v.Check(int2 > 0, "int2", "must be a positive integer")
			v.Check(int2 <= 10000, "int2", "must not be more than 10,000")
			v.Check(int1 != int2, "int1", "must be different from int2")
			v.Check(limit > 0, "limit", "must be a positive integer")
			v.Check(limit <= 100000, "limit", "must not be more than 100,000")
			v.Check(str1 != "", "str1", "must be provided")
			v.Check(len(str1) <= 50, "str1", "must not be more than 50 characters")
			v.Check(str2 != "", "str2", "must be provided")
			v.Check(len(str2) <= 50, "str2", "must not be more than 50 characters")

			_ = v.Valid()
			_ = v.ErrorMap()
		}
	})

	b.Run("Validator-Reuse", func(b *testing.B) {
		v := New()

		for i := 0; i < b.N; i++ {
			v.Clear()

			// Reuse validator for multiple validations
			v.Check(i > 0, "field", "must be positive")
			v.Check(i < 1000000, "field", "must be reasonable")

			_ = v.Valid()
		}
	})
}

// BenchmarkMemoryAllocation tests memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("ErrorMap-Copy-Small", func(b *testing.B) {
		v := New()
		v.AddError("field1", "error1")
		v.AddError("field2", "error2")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = v.ErrorMap()
		}
	})

	b.Run("ErrorMap-Copy-Large", func(b *testing.B) {
		v := New()
		for i := 0; i < 20; i++ {
			v.AddError("field", "error message")
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = v.ErrorMap()
		}
	})

	b.Run("Unique-LargeSlice", func(b *testing.B) {
		// Test performance with larger slices
		values := make([]int, 1000)
		for i := range values {
			values[i] = i
		}

		for i := 0; i < b.N; i++ {
			_ = Unique(values)
		}
	})

	b.Run("PermittedValue-LargeList", func(b *testing.B) {
		// Test performance with larger permitted value lists
		permitted := make([]string, 100)
		for i := range permitted {
			permitted[i] = "value"
		}

		for i := 0; i < b.N; i++ {
			_ = PermittedValue("target", permitted...)
		}
	})
}
