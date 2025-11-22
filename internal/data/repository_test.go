package data

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestStatisticsRepository provides unit tests for the StatisticsRepository interface.
// Uses table-driven tests for comprehensive coverage of all methods.

// MockStatisticsRepository implements StatisticsRepository for unit testing.
// Provides controllable mock implementation without external database dependencies.
type MockStatisticsRepository struct {
	mu           sync.RWMutex
	entries      map[string]*StatisticsEntry
	recordFunc   func(ctx context.Context, input FizzBuzzInput) (*StatisticsEntry, error)
	getMostFunc  func(ctx context.Context) (*StatisticsEntry, error)
	getTopNFunc  func(ctx context.Context, n int) ([]*StatisticsEntry, error)
	getStatsFunc func(ctx context.Context) (StatsSummary, error)
	closeFunc    func() error
}

// NewMockStatisticsRepository creates a new mock repository for testing.
func NewMockStatisticsRepository() *MockStatisticsRepository {
	return &MockStatisticsRepository{
		entries: make(map[string]*StatisticsEntry),
	}
}

// Record implements StatisticsRepository.Record for mock testing.
func (m *MockStatisticsRepository) Record(ctx context.Context, input FizzBuzzInput) (*StatisticsEntry, error) {
	if m.recordFunc != nil {
		return m.recordFunc(ctx, input)
	}

	// Thread-safe mock behavior
	m.mu.Lock()
	defer m.mu.Unlock()

	hash := input.GenerateStatsKey()
	if entry, exists := m.entries[hash]; exists {
		entry.Hits++
		entry.UpdatedAt = time.Now()
		return entry, nil
	}

	entry := &StatisticsEntry{
		ParametersHash: hash,
		Parameters:     input,
		Hits:           1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	m.entries[hash] = entry
	return entry, nil
}

// GetMostFrequent implements StatisticsRepository.GetMostFrequent for mock testing.
func (m *MockStatisticsRepository) GetMostFrequent(ctx context.Context) (*StatisticsEntry, error) {
	if m.getMostFunc != nil {
		return m.getMostFunc(ctx)
	}

	// Thread-safe mock behavior
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.entries) == 0 {
		return nil, nil
	}

	var mostFrequent *StatisticsEntry
	for _, entry := range m.entries {
		if mostFrequent == nil || entry.Hits > mostFrequent.Hits {
			mostFrequent = entry
		}
	}
	return mostFrequent, nil
}

// GetTopN implements StatisticsRepository.GetTopN for mock testing.
func (m *MockStatisticsRepository) GetTopN(ctx context.Context, n int) ([]*StatisticsEntry, error) {
	if m.getTopNFunc != nil {
		return m.getTopNFunc(ctx, n)
	}

	// Thread-safe mock behavior
	m.mu.RLock()
	defer m.mu.RUnlock()

	var entries []*StatisticsEntry
	for _, entry := range m.entries {
		entries = append(entries, entry)
	}

	// Simple selection of first n entries (real implementation would sort)
	if n > len(entries) {
		n = len(entries)
	}
	return entries[:n], nil
}

// GetStats implements StatisticsRepository.GetStats for mock testing.
func (m *MockStatisticsRepository) GetStats(ctx context.Context) (StatsSummary, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx)
	}

	// Thread-safe mock behavior
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := StatsSummary{
		TotalUniqueRequests: int64(len(m.entries)),
	}

	var totalHits int64
	var maxHits int64
	for _, entry := range m.entries {
		totalHits += int64(entry.Hits)
		if int64(entry.Hits) > maxHits {
			maxHits = int64(entry.Hits)
		}
	}

	summary.TotalRequests = totalHits
	summary.MaxHits = maxHits
	if summary.TotalUniqueRequests > 0 {
		summary.AvgHitsPerUniqueRequest = float64(totalHits) / float64(summary.TotalUniqueRequests)
	}

	return summary, nil
}

// Close implements StatisticsRepository.Close for mock testing.
func (m *MockStatisticsRepository) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// Test scenarios for StatisticsRepository interface
func TestStatisticsRepository(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T, repo StatisticsRepository)
	}{
		{"RecordNewEntry", testRecordNewEntry},
		{"RecordExistingEntry", testRecordExistingEntry},
		{"GetMostFrequentEmpty", testGetMostFrequentEmpty},
		{"GetMostFrequentSingle", testGetMostFrequentSingle},
		{"GetMostFrequentMultiple", testGetMostFrequentMultiple},
		{"GetTopNEmpty", testGetTopNEmpty},
		{"GetTopNLimitLargerThanResults", testGetTopNLimitLargerThanResults},
		{"GetStatsEmpty", testGetStatsEmpty},
		{"GetStatsWithData", testGetStatsWithData},
		{"ConcurrentAccess", testConcurrentAccess},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockStatisticsRepository()
			tt.test(t, repo)
		})
	}
}

func testRecordNewEntry(t *testing.T, repo StatisticsRepository) {
	ctx := context.Background()
	input := FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	entry, err := repo.Record(ctx, input)
	if err != nil {
		t.Fatalf("Record() failed: %v", err)
	}

	if entry == nil {
		t.Fatal("Record() returned nil entry")
	}

	if entry.Hits != 1 {
		t.Errorf("Expected hits = 1, got %d", entry.Hits)
	}

	if entry.Parameters != input {
		t.Errorf("Parameters mismatch: expected %v, got %v", input, entry.Parameters)
	}
}

func testRecordExistingEntry(t *testing.T, repo StatisticsRepository) {
	ctx := context.Background()
	input := FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	// Record first time
	_, err := repo.Record(ctx, input)
	if err != nil {
		t.Fatalf("First Record() failed: %v", err)
	}

	// Record second time
	entry2, err := repo.Record(ctx, input)
	if err != nil {
		t.Fatalf("Second Record() failed: %v", err)
	}

	if entry2.Hits != 2 {
		t.Errorf("Expected hits = 2, got %d", entry2.Hits)
	}
}

func testGetMostFrequentEmpty(t *testing.T, repo StatisticsRepository) {
	ctx := context.Background()

	entry, err := repo.GetMostFrequent(ctx)
	if err != nil {
		t.Fatalf("GetMostFrequent() failed: %v", err)
	}

	if entry != nil {
		t.Error("Expected nil for empty repository")
	}
}

func testGetMostFrequentSingle(t *testing.T, repo StatisticsRepository) {
	ctx := context.Background()
	input := FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	recorded, err := repo.Record(ctx, input)
	if err != nil {
		t.Fatalf("Record() failed: %v", err)
	}

	most, err := repo.GetMostFrequent(ctx)
	if err != nil {
		t.Fatalf("GetMostFrequent() failed: %v", err)
	}

	if most == nil {
		t.Fatal("GetMostFrequent() returned nil")
	}

	if most.Parameters != recorded.Parameters {
		t.Errorf("Parameters mismatch: expected %v, got %v", recorded.Parameters, most.Parameters)
	}
}

func testGetMostFrequentMultiple(t *testing.T, repo StatisticsRepository) {
	ctx := context.Background()

	input1 := FizzBuzzInput{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}
	input2 := FizzBuzzInput{Int1: 2, Int2: 7, Limit: 20, Str1: "foo", Str2: "bar"}

	// Record input1 once
	_, err := repo.Record(ctx, input1)
	if err != nil {
		t.Fatalf("Record input1 failed: %v", err)
	}

	// Record input2 twice
	_, err = repo.Record(ctx, input2)
	if err != nil {
		t.Fatalf("Record input2 first time failed: %v", err)
	}
	_, err = repo.Record(ctx, input2)
	if err != nil {
		t.Fatalf("Record input2 second time failed: %v", err)
	}

	most, err := repo.GetMostFrequent(ctx)
	if err != nil {
		t.Fatalf("GetMostFrequent() failed: %v", err)
	}

	if most == nil {
		t.Fatal("GetMostFrequent() returned nil")
	}

	// input2 should be most frequent with 2 hits
	if most.Parameters != input2 {
		t.Errorf("Expected most frequent to be input2, got %v", most.Parameters)
	}
}

func testGetTopNEmpty(t *testing.T, repo StatisticsRepository) {
	ctx := context.Background()

	entries, err := repo.GetTopN(ctx, 5)
	if err != nil {
		t.Fatalf("GetTopN() failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected empty slice, got %d entries", len(entries))
	}
}

func testGetTopNLimitLargerThanResults(t *testing.T, repo StatisticsRepository) {
	ctx := context.Background()
	input := FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	_, err := repo.Record(ctx, input)
	if err != nil {
		t.Fatalf("Record() failed: %v", err)
	}

	entries, err := repo.GetTopN(ctx, 10)
	if err != nil {
		t.Fatalf("GetTopN() failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
}

func testGetStatsEmpty(t *testing.T, repo StatisticsRepository) {
	ctx := context.Background()

	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats() failed: %v", err)
	}

	if stats.TotalUniqueRequests != 0 {
		t.Errorf("Expected 0 unique requests, got %d", stats.TotalUniqueRequests)
	}

	if stats.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests, got %d", stats.TotalRequests)
	}
}

func testGetStatsWithData(t *testing.T, repo StatisticsRepository) {
	ctx := context.Background()
	input := FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	// Record twice
	_, err := repo.Record(ctx, input)
	if err != nil {
		t.Fatalf("First Record() failed: %v", err)
	}
	_, err = repo.Record(ctx, input)
	if err != nil {
		t.Fatalf("Second Record() failed: %v", err)
	}

	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats() failed: %v", err)
	}

	if stats.TotalUniqueRequests != 1 {
		t.Errorf("Expected 1 unique request, got %d", stats.TotalUniqueRequests)
	}

	if stats.TotalRequests != 2 {
		t.Errorf("Expected 2 total requests, got %d", stats.TotalRequests)
	}

	if stats.MaxHits != 2 {
		t.Errorf("Expected max hits = 2, got %d", stats.MaxHits)
	}
}

func testConcurrentAccess(t *testing.T, repo StatisticsRepository) {
	ctx := context.Background()
	input := FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	// Test concurrent recording
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := repo.Record(ctx, input)
			if err != nil {
				t.Errorf("Concurrent Record() failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	most, err := repo.GetMostFrequent(ctx)
	if err != nil {
		t.Fatalf("GetMostFrequent() failed: %v", err)
	}

	if most == nil {
		t.Fatal("GetMostFrequent() returned nil after concurrent access")
	}

	// Note: Mock implementation might not handle concurrency perfectly,
	// but this test ensures the interface supports concurrent usage
}

// Benchmark tests for performance validation
func BenchmarkStatisticsRepository_Record(b *testing.B) {
	repo := NewMockStatisticsRepository()
	ctx := context.Background()
	input := FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := repo.Record(ctx, input)
			if err != nil {
				b.Fatalf("Record() failed: %v", err)
			}
		}
	})
}

func BenchmarkStatisticsRepository_GetMostFrequent(b *testing.B) {
	repo := NewMockStatisticsRepository()
	ctx := context.Background()
	input := FizzBuzzInput{
		Int1:  3,
		Int2:  5,
		Limit: 15,
		Str1:  "fizz",
		Str2:  "buzz",
	}

	// Setup data
	for i := 0; i < 100; i++ {
		_, err := repo.Record(ctx, input)
		if err != nil {
			b.Fatalf("Setup Record() failed: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := repo.GetMostFrequent(ctx)
			if err != nil {
				b.Fatalf("GetMostFrequent() failed: %v", err)
			}
		}
	})
}

// Compile-time verification that MockStatisticsRepository implements StatisticsRepository
var _ StatisticsRepository = (*MockStatisticsRepository)(nil)
