package generator

import (
	"math/rand"
	"time"
)

// SeededRand wraps a deterministic random number generator seeded from a timestamp
type SeededRand struct {
	rng *rand.Rand
}

// NewSeededRand creates a new deterministic RNG from a timestamp
func NewSeededRand(timestamp time.Time) *SeededRand {
	seed := timestamp.UnixNano()
	source := rand.NewSource(seed)
	return &SeededRand{
		rng: rand.New(source),
	}
}

// IntRange returns a random integer in the range [min, max] inclusive
func (sr *SeededRand) IntRange(min, max int) int {
	if min > max {
		min, max = max, min
	}
	return min + sr.rng.Intn(max-min+1)
}

// Choose returns a random element from the provided slice
func (sr *SeededRand) Choose(options []string) string {
	if len(options) == 0 {
		return ""
	}
	idx := sr.rng.Intn(len(options))
	return options[idx]
}

// Shuffle randomizes the order of elements in a slice
func (sr *SeededRand) Shuffle(slice []interface{}) {
	sr.rng.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}

// Float64 returns a random float64 in the range [0.0, 1.0)
func (sr *SeededRand) Float64() float64 {
	return sr.rng.Float64()
}

// Intn returns a random integer in the range [0, n)
func (sr *SeededRand) Intn(n int) int {
	return sr.rng.Intn(n)
}
