package kvstore

import "sync"

type Store struct {
	mu sync.RWMutex
	m  map[string]string
}

var (
	instance *Store
	once     sync.Once
)

func Get() *Store {
	once.Do(func() {
		instance = &Store{
			m: make(map[string]string),
		}
	})
	return instance
}

func (s *Store) GetValue(key string) (string, bool) {
	s.mu.RLock()
	v, ok := s.m[key]
	s.mu.RUnlock()
	return v, ok
}

func (s *Store) SetValue(key, value string) {
	s.mu.Lock()
	s.m[key] = value
	s.mu.Unlock()
}

func (s *Store) Delete(key string) {
	s.mu.Lock()
	delete(s.m, key)
	s.mu.Unlock()
}

func (s *Store) Exists(key string) bool {
	s.mu.RLock()
	_, ok := s.m[key]
	s.mu.RUnlock()
	return ok
}

func (s *Store) Snapshot() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp := make(map[string]string, len(s.m))
	for k, v := range s.m {
		cp[k] = v
	}
	return cp
}
