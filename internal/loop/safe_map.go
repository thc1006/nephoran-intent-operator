package loop

import "sync"

// SafeSet provides a thread-safe set implementation.
type SafeSet struct {
	mu    sync.RWMutex
	items map[string]struct{}
}

// NewSafeSet creates a new thread-safe set.
func NewSafeSet() *SafeSet {
	return &SafeSet{
		items: make(map[string]struct{}),
	}
}

// Add adds an item to the set.
func (s *SafeSet) Add(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items == nil {
		s.items = make(map[string]struct{})
	}
	s.items[key] = struct{}{}
}

// Has checks if an item exists in the set.
func (s *SafeSet) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.items == nil {
		return false
	}
	_, exists := s.items[key]
	return exists
}

// Delete removes an item from the set.
func (s *SafeSet) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items == nil {
		return
	}
	delete(s.items, key)
}

// LoadFromSlice loads items from a slice.
func (s *SafeSet) LoadFromSlice(items []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items == nil {
		s.items = make(map[string]struct{})
	}
	for _, item := range items {
		s.items[item] = struct{}{}
	}
}

// ToSlice returns all items as a slice.
func (s *SafeSet) ToSlice() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.items == nil {
		return []string{}
	}
	result := make([]string, 0, len(s.items))
	for key := range s.items {
		result = append(result, key)
	}
	return result
}

// Size returns the number of items in the set.
func (s *SafeSet) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.items == nil {
		return 0
	}
	return len(s.items)
}
