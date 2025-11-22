package data

import (
	"testing"
)

// BenchmarkStatisticsTracker_Record benchmarks the Record operation.
// Target: <100μs per operation as specified in requirements.
func BenchmarkStatisticsTracker_Record(b *testing.B) {
	tracker := NewStatisticsTracker()
	input := &FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tracker.Record(input)
	}
}

// BenchmarkStatisticsTracker_RecordUnique benchmarks recording unique parameters.
// Tests performance when creating new entries vs incrementing existing ones.
func BenchmarkStatisticsTracker_RecordUnique(b *testing.B) {
	tracker := NewStatisticsTracker()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		input := &FizzBuzzInput{
			Int1:  i % 1000,       // Vary Int1 to create unique combinations
			Int2:  (i % 1000) + 1, // Ensure Int2 is different from Int1
			Limit: 15,
			Str1:  "fizz",
			Str2:  "buzz",
		}
		tracker.Record(input)
	}
}

// BenchmarkStatisticsTracker_GetMostFrequent benchmarks the GetMostFrequent operation.
// Target: <1ms per operation as specified in requirements.
func BenchmarkStatisticsTracker_GetMostFrequent(b *testing.B) {
	tracker := NewStatisticsTracker()

	// Pre-populate tracker with some data
	for i := 0; i < 1000; i++ {
		input := &FizzBuzzInput{
			Int1:  i % 10,
			Int2:  (i % 10) + 1,
			Limit: 15,
			Str1:  "fizz",
			Str2:  "buzz",
		}
		tracker.Record(input)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tracker.GetMostFrequent()
	}
}

// BenchmarkStatisticsTracker_GetMostFrequentLarge benchmarks GetMostFrequent with larger dataset.
// Tests performance scalability with many tracked entries.
func BenchmarkStatisticsTracker_GetMostFrequentLarge(b *testing.B) {
	tracker := NewStatisticsTracker()

	// Pre-populate tracker with larger dataset
	for i := 0; i < 10000; i++ {
		input := &FizzBuzzInput{
			Int1:  i % 100,
			Int2:  (i % 100) + 1,
			Limit: 15,
			Str1:  "fizz",
			Str2:  "buzz",
		}
		tracker.Record(input)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tracker.GetMostFrequent()
	}
}

// BenchmarkStatisticsTracker_MixedOperations benchmarks realistic mixed workload.
// Simulates production usage with both reads and writes.
func BenchmarkStatisticsTracker_MixedOperations(b *testing.B) {
	tracker := NewStatisticsTracker()
	input := &FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 80% reads, 20% writes (typical read-heavy workload)
		if i%5 == 0 {
			tracker.Record(input)
		} else {
			tracker.GetMostFrequent()
		}
	}
}

// BenchmarkStatisticsTracker_KeyGeneration benchmarks key generation performance.
// Tests the overhead of SHA256 hash generation for parameter keys.
func BenchmarkStatisticsTracker_KeyGeneration(b *testing.B) {
	input := &FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		input.GenerateStatsKey()
	}
}

// BenchmarkStatisticsTracker_GetStats benchmarks stats retrieval.
// Tests performance of copying all statistics data.
func BenchmarkStatisticsTracker_GetStats(b *testing.B) {
	tracker := NewStatisticsTracker()

	// Pre-populate with moderate dataset
	for i := 0; i < 1000; i++ {
		input := &FizzBuzzInput{
			Int1:  i % 50,
			Int2:  (i % 50) + 1,
			Limit: 15,
			Str1:  "fizz",
			Str2:  "buzz",
		}
		tracker.Record(input)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tracker.GetStats()
	}
}

// BenchmarkStatisticsTracker_EntryCount benchmarks entry counting.
// Tests simple map length operation performance.
func BenchmarkStatisticsTracker_EntryCount(b *testing.B) {
	tracker := NewStatisticsTracker()

	// Pre-populate with some data
	for i := 0; i < 100; i++ {
		input := &FizzBuzzInput{
			Int1:  i % 10,
			Int2:  (i % 10) + 1,
			Limit: 15,
			Str1:  "fizz",
			Str2:  "buzz",
		}
		tracker.Record(input)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tracker.EntryCount()
	}
}
