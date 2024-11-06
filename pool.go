package g

import (
	"sync"
)

type Pool[T any] struct {
	pool   *sync.Pool
	getter func(value T)
	putter func(value T)
}

func NewPool[T any](
	create func() T,
	getter func(value T),
	putter func(value T),
) *Pool[T] {
	if create == nil {
		panic("create must not be nil")
	}
	return &Pool[T]{
		pool: &sync.Pool{
			New: func() interface{} {
				return create()
			},
		},
		getter: getter,
		putter: putter,
	}
}

func (s *Pool[T]) Get() T {
	value := s.pool.Get().(T)
	if s.getter != nil {
		s.getter(value)
	}
	return value
}

func (s *Pool[T]) Put(value T) {
	if s.putter != nil {
		s.putter(value)
	}
	s.pool.Put(value)
}
