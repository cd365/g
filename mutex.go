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

func (s *Mutex) Locker() sync.Locker {
	return s.mutex
}

func (s *Mutex) WithLock(fc func()) {
	if fc != nil {
		s.Lock()
		defer s.Unlock() // in case `call` panics
		fc()
	}
}

type RWMutex struct {
	mutex *sync.RWMutex
}

func NewRWMutex() *RWMutex {
	return &RWMutex{
		mutex: &sync.RWMutex{},
	}
}

func (s *RWMutex) Lock() {
	s.mutex.Lock()
}

func (s *RWMutex) RLock() {
	s.mutex.RLock()
}

func (s *RWMutex) RLocker() sync.Locker {
	return s.mutex.RLocker()
}

func (s *RWMutex) RUnlock() {
	s.mutex.RUnlock()
}

func (s *RWMutex) TryLock() bool {
	return s.mutex.TryLock()
}

func (s *RWMutex) TryRLock() bool {
	return s.mutex.TryRLock()
}

func (s *RWMutex) Unlock() {
	s.mutex.Unlock()
}

func (s *RWMutex) Locker() sync.Locker {
	return s.mutex
}

func (s *RWMutex) WithLock(fc func()) {
	if fc != nil {
		s.Lock()
		defer s.Unlock() // in case `call` panics
		fc()
	}
}

func (s *RWMutex) WithRLock(fc func()) {
	if fc != nil {
		s.RLock()
		defer s.RUnlock() // in case `call` panics
		fc()
	}
}
