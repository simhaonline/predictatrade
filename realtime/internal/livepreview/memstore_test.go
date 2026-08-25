package livepreview

import (
	"sync"
	"time"
)

// MemStore is an in-memory store for unit tests.
type MemStore struct {
	mu     sync.Mutex
	trials map[string]*Trial
	events map[string][]string
}

func NewMemStore() *MemStore {
	return &MemStore{trials: map[string]*Trial{}, events: map[string][]string{}}
}

func (m *MemStore) GetByTokenHash(hash string) (*Trial, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.trials[hash]
	if !ok {
		return nil, nil
	}
	return t, nil  // return pointer directly — avoids copying Mutex
}

func (m *MemStore) Insert(t *Trial) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trials[t.TokenHash] = t  // store pointer directly — avoids copying Mutex
	return nil
}

func (m *MemStore) Save(t *Trial) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trials[t.TokenHash] = t  // store pointer directly — avoids copying Mutex
	return nil
}

func (m *MemStore) CountRecent(ipHash, uaHash string, since time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, t := range m.trials {
		if t.StartedAt.After(since) && t.IPHash == ipHash && t.UAHash == uaHash {
			n++
		}
	}
	return n, nil
}

func (m *MemStore) RecordEvent(tokenHash, event string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events[tokenHash] = append(m.events[tokenHash], event)
	return nil
}

func (m *MemStore) Events(tokenHash string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.events[tokenHash]))
	copy(out, m.events[tokenHash])
	return out
}

func (m *MemStore) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.trials)
}
