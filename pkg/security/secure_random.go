// Package security provides cryptographically secure random number generation utilities
// replacing all instances of insecure math/rand usage across the Nephoran codebase
package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"math/big"
	"sync"
	"time"
)

// SecureRandom provides cryptographically secure random number generation
// This replaces all insecure math/rand usage throughout the codebase
type SecureRandom struct {
	reader io.Reader
	mu     sync.Mutex
}

// Global secure random instance for package-level functions
var globalSecureRandom = NewSecureRandom()

// NewSecureRandom creates a new cryptographically secure random number generator
func NewSecureRandom() *SecureRandom {
	return &SecureRandom{
		reader: rand.Reader,
	}
}

// Int63 returns a non-negative pseudo-random 63-bit integer as an int64
// This is a drop-in replacement for math/rand.Int63()
func (sr *SecureRandom) Int63() int64 {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	// Generate 8 bytes and mask to 63 bits
	b := make([]byte, 8)
	if _, err := io.ReadFull(sr.reader, b); err != nil {
		panic(fmt.Sprintf("secure random generation failed: %v", err))
	}
	
	// Convert to int64 and mask to 63 bits (remove sign bit)
	return int64(uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])) & 0x7fffffffffffffff
}

// Intn returns, as an int, a non-negative pseudo-random number in [0,n)
// This is a drop-in replacement for math/rand.Intn()
func (sr *SecureRandom) Intn(n int) int {
	if n <= 0 {
		panic("invalid argument to Intn")
	}
	
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	// Use crypto/rand to generate a big.Int in the range [0, n)
	nBig := big.NewInt(int64(n))
	result, err := rand.Int(sr.reader, nBig)
	if err != nil {
		panic(fmt.Sprintf("secure random generation failed: %v", err))
	}
	
	return int(result.Int64())
}

// Int31 returns a non-negative pseudo-random 31-bit integer as an int32
// This is a drop-in replacement for math/rand.Int31()
func (sr *SecureRandom) Int31() int32 {
	return int32(sr.Int63() >> 32)
}

// Int31n returns, as an int32, a non-negative pseudo-random number in [0,n)
// This is a drop-in replacement for math/rand.Int31n()
func (sr *SecureRandom) Int31n(n int32) int32 {
	if n <= 0 {
		panic("invalid argument to Int31n")
	}
	return int32(sr.Intn(int(n)))
}

// Int returns a non-negative pseudo-random int
// This is a drop-in replacement for math/rand.Int()
func (sr *SecureRandom) Int() int {
	return int(sr.Int63())
}

// Float64 returns, as a float64, a pseudo-random number in [0.0,1.0)
// This is a drop-in replacement for math/rand.Float64()
func (sr *SecureRandom) Float64() float64 {
	// Generate 53 bits of precision (mantissa of float64)
	val := sr.Int63() >> 11  // Use top 53 bits
	return float64(val) / (1 << 53)
}

// Float32 returns, as a float32, a pseudo-random number in [0.0,1.0)
// This is a drop-in replacement for math/rand.Float32()
func (sr *SecureRandom) Float32() float32 {
	// Generate 24 bits of precision (mantissa of float32)
	val := sr.Int31() >> 8   // Use top 24 bits
	return float32(val) / (1 << 24)
}

// Duration returns a secure random duration between min and max
func (sr *SecureRandom) Duration(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}
	diff := max - min
	random := sr.Int63n(int64(diff))
	return min + time.Duration(random)
}

// Int63n returns, as an int64, a non-negative pseudo-random number in [0,n)
func (sr *SecureRandom) Int63n(n int64) int64 {
	if n <= 0 {
		panic("invalid argument to Int63n")
	}
	
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	nBig := big.NewInt(n)
	result, err := rand.Int(sr.reader, nBig)
	if err != nil {
		panic(fmt.Sprintf("secure random generation failed: %v", err))
	}
	
	return result.Int64()
}

// Perm returns, as a slice of n ints, a pseudo-random permutation of the integers [0,n)
// This is a drop-in replacement for math/rand.Perm()
func (sr *SecureRandom) Perm(n int) []int {
	if n < 0 {
		panic("invalid argument to Perm")
	}
	
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	
	// Fisher-Yates shuffle using secure random
	for i := n - 1; i > 0; i-- {
		j := sr.Intn(i + 1)
		p[i], p[j] = p[j], p[i]
	}
	
	return p
}

// Shuffle pseudo-randomizes the order of elements using secure random
// This is a drop-in replacement for math/rand.Shuffle()
func (sr *SecureRandom) Shuffle(n int, swap func(i, j int)) {
	if n < 0 {
		panic("invalid argument to Shuffle")
	}
	
	// Fisher-Yates shuffle
	for i := n - 1; i > 0; i-- {
		j := sr.Intn(i + 1)
		swap(i, j)
	}
}

// ExpFloat64 returns an exponentially distributed float64
func (sr *SecureRandom) ExpFloat64() float64 {
	for {
		u := sr.Float64()
		if u != 0 {
			return -math.Log(u)
		}
	}
}

// NormFloat64 returns a normally distributed float64 in the range [-math.MaxFloat64, +math.MaxFloat64]
func (sr *SecureRandom) NormFloat64() float64 {
	// Box-Muller transform
	for {
		u := sr.Float64()
		v := sr.Float64()
		if u != 0 {
			z := math.Sqrt(-2*math.Log(u)) * math.Cos(2*math.Pi*v)
			return z
		}
	}
}

// Bytes fills the provided byte slice with secure random bytes
func (sr *SecureRandom) Bytes(b []byte) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	if _, err := io.ReadFull(sr.reader, b); err != nil {
		panic(fmt.Sprintf("secure random generation failed: %v", err))
	}
}

// SecureToken generates a cryptographically secure token of specified length
func (sr *SecureRandom) SecureToken(length int) string {
	bytes := make([]byte, length)
	sr.Bytes(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// SecureID generates a cryptographically secure ID (32 bytes = 256 bits)
func (sr *SecureRandom) SecureID() string {
	return sr.SecureToken(32)
}

// SecureSessionID generates a secure session ID
func (sr *SecureRandom) SecureSessionID() string {
	return sr.SecureToken(24) // 192 bits
}

// Package-level convenience functions using the global secure random instance

// Int63 returns a secure random 63-bit integer
func Int63() int64 {
	return globalSecureRandom.Int63()
}

// Intn returns a secure random number in [0,n)
func Intn(n int) int {
	return globalSecureRandom.Intn(n)
}

// Int31 returns a secure random 31-bit integer
func Int31() int32 {
	return globalSecureRandom.Int31()
}

// Int31n returns a secure random number in [0,n)
func Int31n(n int32) int32 {
	return globalSecureRandom.Int31n(n)
}

// Int returns a secure random int
func Int() int {
	return globalSecureRandom.Int()
}

// Float64 returns a secure random float64 in [0.0,1.0)
func Float64() float64 {
	return globalSecureRandom.Float64()
}

// Float32 returns a secure random float32 in [0.0,1.0)
func Float32() float32 {
	return globalSecureRandom.Float32()
}

// Duration returns a secure random duration between min and max
func Duration(min, max time.Duration) time.Duration {
	return globalSecureRandom.Duration(min, max)
}

// Int63n returns a secure random number in [0,n)
func Int63n(n int64) int64 {
	return globalSecureRandom.Int63n(n)
}

// Perm returns a secure random permutation of [0,n)
func Perm(n int) []int {
	return globalSecureRandom.Perm(n)
}

// Shuffle securely randomizes the order of elements
func Shuffle(n int, swap func(i, j int)) {
	globalSecureRandom.Shuffle(n, swap)
}

// ExpFloat64 returns a secure exponentially distributed float64
func ExpFloat64() float64 {
	return globalSecureRandom.ExpFloat64()
}

// NormFloat64 returns a secure normally distributed float64
func NormFloat64() float64 {
	return globalSecureRandom.NormFloat64()
}

// Bytes fills the slice with secure random bytes
func Bytes(b []byte) {
	globalSecureRandom.Bytes(b)
}

// SecureToken generates a cryptographically secure token
func SecureToken(length int) string {
	return globalSecureRandom.SecureToken(length)
}

// SecureID generates a cryptographically secure ID
func SecureID() string {
	return globalSecureRandom.SecureID()
}

// SecureSessionID generates a secure session ID
func SecureSessionID() string {
	return globalSecureRandom.SecureSessionID()
}

// Helper functions for specific use cases

// SecureJitter adds cryptographically secure jitter to a base duration
func SecureJitter(base time.Duration, jitterPercent float64) time.Duration {
	if jitterPercent <= 0 || jitterPercent > 1 {
		return base
	}
	
	maxJitter := time.Duration(float64(base) * jitterPercent)
	jitter := Duration(-maxJitter, maxJitter)
	return base + jitter
}

// SecureBackoff calculates exponential backoff with secure jitter
func SecureBackoff(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	
	// Calculate exponential backoff
	delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
	
	// Cap at max delay
	if delay > maxDelay {
		delay = maxDelay
	}
	
	// Add secure jitter (±25%)
	return SecureJitter(delay, 0.25)
}

// SecureChoice randomly selects an element from a slice using secure random
func SecureChoice[T any](items []T) T {
	if len(items) == 0 {
		var zero T
		return zero
	}
	
	index := Intn(len(items))
	return items[index]
}

// SecureWeightedChoice randomly selects an element based on weights using secure random
func SecureWeightedChoice[T any](items []T, weights []int) T {
	if len(items) == 0 || len(items) != len(weights) {
		var zero T
		return zero
	}
	
	// Calculate total weight
	totalWeight := 0
	for _, w := range weights {
		totalWeight += w
	}
	
	if totalWeight <= 0 {
		var zero T
		return zero
	}
	
	// Generate random number and select based on cumulative weights
	r := Intn(totalWeight)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return items[i]
		}
	}
	
	// Fallback to last item (shouldn't happen)
	return items[len(items)-1]
}

// ValidateSecureRandomness validates that the random number generator is working correctly
func ValidateSecureRandomness() error {
	// Test basic functionality
	for i := 0; i < 100; i++ {
		val := Float64()
		if val < 0 || val >= 1 {
			return fmt.Errorf("Float64() returned %f, expected [0,1)", val)
		}
	}
	
	// Test randomness quality (basic chi-square test)
	buckets := make([]int, 10)
	samples := 10000
	
	for i := 0; i < samples; i++ {
		val := Intn(10)
		buckets[val]++
	}
	
	expected := samples / 10
	chiSquare := 0.0
	
	for _, count := range buckets {
		diff := float64(count - expected)
		chiSquare += (diff * diff) / float64(expected)
	}
	
	// With 9 degrees of freedom, chi-square should be less than 27.88 for 99% confidence
	if chiSquare > 27.88 {
		return fmt.Errorf("randomness test failed: chi-square = %f (too high)", chiSquare)
	}
	
	return nil
}