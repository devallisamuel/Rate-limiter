package limiter

import (
	"context"
	"sync"
	"time"
)

type Store struct {
	mu      sync.Mutex // guards clients
	clients map[string]Allower

	factory func() Allower

	now func() time.Time
}

func NewStore(limit int, window time.Duration) *Store {
	return NewStoreWith(func() Allower { return New(limit, window) })
}

func NewStoreWith(factory func() Allower) *Store {
	return newStore(factory, time.Now)
}

func newStore(factory func() Allower, now func() time.Time) *Store {
	return &Store{
		clients: map[string]Allower{},
		factory: factory,
		now:     now,
	}
}

func (s *Store) Get(key string) Allower {
	s.mu.Lock()
	defer s.mu.Unlock()

	if a, ok := s.clients[key]; ok {
		return a
	}

	a := s.factory()
	s.clients[key] = a
	return a
}

func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}

func (s *Store) removeIdle(idle time.Duration) {
	now := s.now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, a := range s.clients {

		if now.Sub(a.LastSeen()) > idle {
			delete(s.clients, key)
		}
	}
}

func (s *Store) Cleanup(ctx context.Context, every, idle time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop() // runs when we return, i.e. after ctx is cancelled

	for {
		select {
		case <-ticker.C:
			s.removeIdle(idle)
		case <-ctx.Done(): // closed when the caller calls cancel()
			return
		}
	}
}
