package g

import (
	"sync"
)

type Mutex struct {
	mutex *sync.Mutex
}

func NewMutex() *Mutex {
	return &Mutex{
		mutex: &sync.Mutex{},
	}
}

func (s *Mutex) Lock() {
	s.mutex.Lock()
}

func (s *Mutex) TryLock() bool {
	return s.mutex.TryLock()
}

func (s *Mutex) Unlock() {
	s.mutex.Unlock()
}

func (s *Mutex) WithLock(fn func()) {
	if fn != nil {
		s.Lock()
		defer s.Unlock() // in case `call` panics
		fn()
	}
}
