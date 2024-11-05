package g

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"sync"
)

const (
	memoryMapCapacity = 32
)

type Memory[V interface{}] struct {
	count  int
	memory []map[string]V
	mutex  []*sync.RWMutex
}

func NewMemory[V interface{}](count int) *Memory[V] {
	if count < 1 || count > 1000 {
		count = 10
	}
	s := &Memory[V]{
		count: count,
	}
	s.memory = make([]map[string]V, s.count)
	s.mutex = make([]*sync.RWMutex, s.count)
	for i := 0; i < s.count; i++ {
		s.mutex[i] = &sync.RWMutex{}
		s.memory[i] = make(map[string]V, memoryMapCapacity)
	}
	return s
}

func (s *Memory[V]) Count() int {
	return s.count
}

func (s *Memory[V]) Index(key string, count int) int {
	hash := sha256.New()
	hash.Write([]byte(key))
	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	hashString016 := hashString[0:8]
	num, err := strconv.ParseInt(hashString016, 16, 64)
	Err(err)
	return int(num % int64(count))
}

func (s *Memory[V]) Get(key string) (value V, exists bool) {
	index := s.Index(key, s.count)
	locker := s.mutex[index]
	locker.RLock()
	defer locker.RUnlock()
	value, exists = s.memory[index][key]
	return
}

func (s *Memory[V]) Set(key string, value V) {
	index := s.Index(key, s.count)
	locker := s.mutex[index]
	locker.Lock()
	defer locker.Unlock()
	s.memory[index][key] = value
}

func (s *Memory[V]) Del(key string) {
	index := s.Index(key, s.count)
	locker := s.mutex[index]
	locker.Lock()
	defer locker.Unlock()
	delete(s.memory[index], key)
}

type node struct {
	index int
	value interface{}
}

func (s *Memory[V]) All() []map[string]V {
	result := make([]map[string]V, s.count)
	lists := make(chan *node, s.count)
	defer close(lists)
	wg := &sync.WaitGroup{}
	for i := 0; i < s.count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			s.mutex[index].RLock()
			defer s.mutex[index].RUnlock()
			tmp := make(map[string]V, len(s.memory[index]))
			for k, v := range s.memory[index] {
				tmp[k] = v
			}
			lists <- &node{
				index: index,
				value: tmp,
			}
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < s.count; i++ {
			tmp := <-lists
			result[tmp.index] = tmp.value.(map[string]V)
		}
	}()
	wg.Wait()
	return result
}

func (s *Memory[V]) Clean() {
	wg := &sync.WaitGroup{}
	for i := 0; i < s.count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			s.mutex[index].Lock()
			defer s.mutex[index].Unlock()
			s.memory[index] = make(map[string]V, memoryMapCapacity)
		}(i)
	}
	wg.Wait()
}

func (s *Memory[V]) Range(fc func(index int, key string, value V)) {
	wg := &sync.WaitGroup{}
	for i := 0; i < s.count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			s.mutex[index].RLock()
			defer s.mutex[index].RUnlock()
			for k, v := range s.memory[index] {
				fc(index, k, v)
			}
		}(i)
	}
	wg.Wait()
}

func (s *Memory[V]) Repair() {
	wg := &sync.WaitGroup{}
	for i := 0; i < s.count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			memory := make(map[string]V, memoryMapCapacity)
			s.mutex[index].Lock()
			defer s.mutex[index].Unlock()
			for k, v := range s.memory[index] {
				memory[k] = v
			}
			s.memory[index] = memory
		}(i)
	}
	wg.Wait()
}
