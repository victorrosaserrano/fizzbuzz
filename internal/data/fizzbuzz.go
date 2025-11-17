package data

import (
	"runtime"
	"strconv"
	"sync"
)

// FizzBuzz generates a sequence of numbers from 1 to the limit where numbers
// divisible by int1 are replaced with str1, numbers divisible by int2 are
// replaced with str2, and numbers divisible by both are replaced with str1+str2.
func FizzBuzz(int1, int2, limit int, str1, str2 string) []string {
	// Pre-allocate slice with capacity for optimal performance
	result := make([]string, 0, limit)

	for i := 1; i <= limit; i++ {
		// Check divisibility by both int1 and int2
		divisibleByInt1 := i%int1 == 0
		divisibleByInt2 := i%int2 == 0

		if divisibleByInt1 && divisibleByInt2 {
			// Divisible by both: concatenate str1+str2
			result = append(result, str1+str2)
		} else if divisibleByInt1 {
			// Divisible only by int1
			result = append(result, str1)
		} else if divisibleByInt2 {
			// Divisible only by int2
			result = append(result, str2)
		} else {
			// Not divisible by either: use string representation of number
			result = append(result, strconv.Itoa(i))
		}
	}

	return result
}

// ConcurrentFizzBuzz generates a FizzBuzz sequence using goroutines for high-volume scenarios.
// Uses a worker pool pattern with intelligent work distribution for optimal performance.
// Best suited for limits > 100,000 where concurrency overhead is justified.
func ConcurrentFizzBuzz(int1, int2, limit int, str1, str2 string) []string {
	if limit <= 0 {
		return []string{}
	}

	// Use the number of CPU cores as a worker count for optimal parallelism
	numWorkers := runtime.NumCPU()
	if numWorkers > limit {
		numWorkers = limit // Don't create more workers than work items
	}

	// Calculate chunk size with proper distribution
	chunkSize := limit / numWorkers
	remainder := limit % numWorkers

	// Pre-allocate result slice and worker result slices
	result := make([]string, limit)
	var wg sync.WaitGroup

	// Launch workers with balanced work distribution
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)

		// Calculate start and end for this worker
		start := w*chunkSize + 1
		end := start + chunkSize - 1

		// The last worker takes any remainder
		if w == numWorkers-1 {
			end += remainder
		}

		go func(workerStart, workerEnd, offset int) {
			defer wg.Done()

			// Process assigned range
			for i := workerStart; i <= workerEnd; i++ {
				divisibleByInt1 := i%int1 == 0
				divisibleByInt2 := i%int2 == 0

				var value string
				if divisibleByInt1 && divisibleByInt2 {
					value = str1 + str2
				} else if divisibleByInt1 {
					value = str1
				} else if divisibleByInt2 {
					value = str2
				} else {
					value = strconv.Itoa(i)
				}

				// Write directly to the final position (i-1 for 0-based indexing)
				result[i-1] = value
			}
		}(start, end, w*chunkSize)
	}

	// Wait for all workers to complete
	wg.Wait()
	return result
}
