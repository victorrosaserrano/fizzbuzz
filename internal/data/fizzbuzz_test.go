package data

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
)

func TestFizzBuzz(t *testing.T) {
	tests := []struct {
		name     string
		int1     int
		int2     int
		limit    int
		str1     string
		str2     string
		expected []string
	}{
		{
			name:     "classic fizzbuzz 3,5,15",
			int1:     3,
			int2:     5,
			limit:    15,
			str1:     "fizz",
			str2:     "buzz",
			expected: []string{"1", "2", "fizz", "4", "buzz", "fizz", "7", "8", "fizz", "buzz", "11", "fizz", "13", "14", "fizzbuzz"},
		},
		{
			name:     "limit 1 not divisible",
			int1:     3,
			int2:     5,
			limit:    1,
			str1:     "fizz",
			str2:     "buzz",
			expected: []string{"1"},
		},
		{
			name:     "limit 1 divisible by int1",
			int1:     1,
			int2:     5,
			limit:    1,
			str1:     "fizz",
			str2:     "buzz",
			expected: []string{"fizz"},
		},
		{
			name:     "limit 1 divisible by both",
			int1:     1,
			int2:     1,
			limit:    1,
			str1:     "fizz",
			str2:     "buzz",
			expected: []string{"fizzbuzz"},
		},
		{
			name:     "different numbers and strings",
			int1:     2,
			int2:     7,
			limit:    14,
			str1:     "ping",
			str2:     "pong",
			expected: []string{"1", "ping", "3", "ping", "5", "ping", "pong", "ping", "9", "ping", "11", "ping", "13", "pingpong"},
		},
		{
			name:     "large int1 and int2 with small limit",
			int1:     100,
			int2:     200,
			limit:    10,
			str1:     "large1",
			str2:     "large2",
			expected: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
		},
		{
			name:     "int1 equals int2",
			int1:     4,
			int2:     4,
			limit:    8,
			str1:     "same",
			str2:     "same",
			expected: []string{"1", "2", "3", "samesame", "5", "6", "7", "samesame"},
		},
		{
			name:     "empty strings",
			int1:     3,
			int2:     5,
			limit:    6,
			str1:     "",
			str2:     "",
			expected: []string{"1", "2", "", "4", "", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FizzBuzz(tt.int1, tt.int2, tt.limit, tt.str1, tt.str2)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("FizzBuzz(%d, %d, %d, %q, %q) = %v, want %v",
					tt.int1, tt.int2, tt.limit, tt.str1, tt.str2, result, tt.expected)
			}

			// Verify slice length
			if len(result) != tt.limit {
				t.Errorf("FizzBuzz result length = %d, want %d", len(result), tt.limit)
			}
		})
	}
}

func TestFizzBuzzEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		int1     int
		int2     int
		limit    int
		str1     string
		str2     string
		validate func(t *testing.T, result []string)
	}{
		{
			name:  "large limit performance test",
			int1:  3,
			int2:  5,
			limit: 10000,
			str1:  "fizz",
			str2:  "buzz",
			validate: func(t *testing.T, result []string) {
				if len(result) != 10000 {
					t.Errorf("Expected length 10000, got %d", len(result))
				}
				// Validate some specific positions
				if result[2] != "fizz" { // position 3
					t.Errorf("Expected fizz at position 3, got %s", result[2])
				}
				if result[4] != "buzz" { // position 5
					t.Errorf("Expected buzz at position 5, got %s", result[4])
				}
				if result[14] != "fizzbuzz" { // position 15
					t.Errorf("Expected fizzbuzz at position 15, got %s", result[14])
				}
			},
		},
		{
			name:  "maximum practical limit",
			int1:  7,
			int2:  11,
			limit: 100000,
			str1:  "seven",
			str2:  "eleven",
			validate: func(t *testing.T, result []string) {
				if len(result) != 100000 {
					t.Errorf("Expected length 100000, got %d", len(result))
				}
				// Validate specific positions
				if result[6] != "seven" { // position 7
					t.Errorf("Expected seven at position 7, got %s", result[6])
				}
				if result[10] != "eleven" { // position 11
					t.Errorf("Expected eleven at position 11, got %s", result[10])
				}
				if result[76] != "seveneleven" { // position 77 (7*11=77)
					t.Errorf("Expected seveneleven at position 77, got %s", result[76])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FizzBuzz(tt.int1, tt.int2, tt.limit, tt.str1, tt.str2)
			tt.validate(t, result)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkFizzBuzz(b *testing.B) {
	benchmarks := []struct {
		name  string
		int1  int
		int2  int
		limit int
		str1  string
		str2  string
	}{
		{"limit_100", 3, 5, 100, "fizz", "buzz"},
		{"limit_1000", 3, 5, 1000, "fizz", "buzz"},
		{"limit_10000", 3, 5, 10000, "fizz", "buzz"},
		{"limit_100000", 3, 5, 100000, "fizz", "buzz"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				FizzBuzz(bm.int1, bm.int2, bm.limit, bm.str1, bm.str2)
			}
		})
	}
}

// Memory allocation benchmark
func BenchmarkFizzBuzzAllocs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		FizzBuzz(3, 5, 1000, "fizz", "buzz")
	}
}

// CONCURRENT VERSION TESTS

func TestConcurrentFizzBuzz(t *testing.T) {
	tests := []struct {
		name  string
		int1  int
		int2  int
		limit int
		str1  string
		str2  string
	}{
		{"classic fizzbuzz 3,5,15", 3, 5, 15, "fizz", "buzz"},
		{"large dataset", 3, 5, 10000, "fizz", "buzz"},
		{"very large dataset", 7, 11, 100000, "seven", "eleven"},
		{"edge case empty", 3, 5, 0, "fizz", "buzz"},
		{"edge case single", 3, 5, 1, "fizz", "buzz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that concurrent version produces same results as sequential
			sequential := FizzBuzz(tt.int1, tt.int2, tt.limit, tt.str1, tt.str2)
			concurrent := ConcurrentFizzBuzz(tt.int1, tt.int2, tt.limit, tt.str1, tt.str2)

			if !reflect.DeepEqual(sequential, concurrent) {
				t.Errorf("ConcurrentFizzBuzz results differ from FizzBuzz")
				t.Logf("Sequential length: %d, Concurrent length: %d", len(sequential), len(concurrent))

				// Show first few differences for debugging
				maxCheck := min(len(sequential), len(concurrent), 20)
				for i := 0; i < maxCheck; i++ {
					if i >= len(sequential) || i >= len(concurrent) {
						break
					}
					if sequential[i] != concurrent[i] {
						t.Logf("Difference at index %d: sequential=%q, concurrent=%q", i, sequential[i], concurrent[i])
					}
				}
			}
		})
	}
}

func TestConcurrentFizzBuzzRaceConditions(t *testing.T) {
	// Test for race conditions by running multiple concurrent operations
	const numGoroutines = 10
	const limit = 1000

	var wg sync.WaitGroup
	results := make([][]string, numGoroutines)

	// Run multiple concurrent FizzBuzz operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = ConcurrentFizzBuzz(3, 5, limit, "fizz", "buzz")
		}(i)
	}

	wg.Wait()

	// Verify all results are identical
	expected := FizzBuzz(3, 5, limit, "fizz", "buzz")
	for i, result := range results {
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Race condition detected: result %d differs from expected", i)
		}
	}
}

// BENCHMARK COMPARISONS

func BenchmarkCompareSequentialVsConcurrent(b *testing.B) {
	benchmarks := []struct {
		name  string
		limit int
	}{
		{"small_1000", 1000},
		{"medium_10000", 10000},
		{"large_100000", 100000},
		{"xlarge_500000", 500000},
		{"xxlarge_1000000", 1000000},
	}

	for _, bm := range benchmarks {
		// Sequential version
		b.Run("Sequential_"+bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				FizzBuzz(3, 5, bm.limit, "fizz", "buzz")
			}
		})

		// Concurrent version
		b.Run("Concurrent_"+bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ConcurrentFizzBuzz(3, 5, bm.limit, "fizz", "buzz")
			}
		})
	}
}

func BenchmarkConcurrentFizzBuzzCPUScaling(b *testing.B) {
	const limit = 1000000

	// Test with different numbers of workers
	cpuCounts := []int{1, 2, 4, 8, runtime.NumCPU()}

	for _, cpus := range cpuCounts {
		b.Run(fmt.Sprintf("CPUs_%d", cpus), func(b *testing.B) {
			oldMaxProcs := runtime.GOMAXPROCS(cpus)
			defer runtime.GOMAXPROCS(oldMaxProcs)

			for i := 0; i < b.N; i++ {
				ConcurrentFizzBuzz(3, 5, limit, "fizz", "buzz")
			}
		})
	}
}

func BenchmarkConcurrentFizzBuzzAllocs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ConcurrentFizzBuzz(3, 5, 100000, "fizz", "buzz")
	}
}

// Helper function for min (Go < 1.21 compatibility)
func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}
