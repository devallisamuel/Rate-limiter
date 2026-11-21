package limiter

import (
	"sync"
	"time"
)

type Store struct {
	mu      sync.Mutex
	clients map[string]*Limiter
	limit   int
	window  time.Duration
}

func NewStore(limit int, window time.Duration) *Store {
	return &Store{
		clients: map[string]*Limiter{},
		limit:   limit,
		window:  window,
	}
}

func (s *Store) Get(key string) *Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()
	// TODO: return the existing limiter, or create, save and return a new one
	hasLimiter, ok := s.clients[key]

	if ok {
		return hasLimiter
	}

	tobeReturned := New(s.limit, s.window)
	s.clients[key] = tobeReturned
	return tobeReturned

}
