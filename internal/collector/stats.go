package collector

import (
	"maps"
	"sync"
)

type CollectionStats struct {
	mu             sync.Mutex
	counts         map[string]int
	totalCollected int
	errors         []error
}

func NewCollectionStats() *CollectionStats {
	return &CollectionStats{
		counts: make(map[string]int),
		errors: make([]error, 0),
	}
}

func (s *CollectionStats) AddCount(resourceType string, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[resourceType] += count
	s.totalCollected += count
}

func (s *CollectionStats) AddError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors = append(s.errors, err)
}

func (s *CollectionStats) GetStats() (map[string]int, int, []error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	counts := make(map[string]int)
	maps.Copy(counts, s.counts)
	return counts, s.totalCollected, append([]error{}, s.errors...)
}