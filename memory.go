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
	count   int
	memory  []map[string]V
	mutex   *RWMutex
	mutexes []*RWMutex
}

func NewMemory[V interface{}](count int) *Memory[V] {
	if count < 1 || count > 1000 {
		count = 10
	}
	s := &Memory[V]{
		count: count,
		mutex: NewRWMutex(),
	}
	s.memory = make([]map[string]V, s.count)
	s.mutexes = make([]*RWMutex, s.count)
	for i := 0; i < s.count; i++ {
		s.memory[i] = make(map[string]V, memoryMapCapacity)
		s.mutexes[i] = NewRWMutex()
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

func (s *Memory[V]) withLocker(locker sync.Locker, fc func()) {
	if locker == nil || fc == nil {
		return
	}
	locker.Lock()
	defer locker.Unlock()
	fc()
}

func (s *Memory[V]) readMemory(index int) (node map[string]V) {
	s.withLocker(s.mutex.RLocker(), func() { node = s.memory[index] })
	return
}

func (s *Memory[V]) writeMemory(index int, node map[string]V) {
	s.withLocker(s.mutex.Locker(), func() { s.memory[index] = node })
}

func (s *Memory[V]) readMemoryNode(index int, fc func()) {
	s.withLocker(s.mutexes[index].RLocker(), fc)
}

func (s *Memory[V]) writeMemoryNode(index int, fc func()) {
	s.withLocker(s.mutexes[index].Locker(), fc)
}

func (s *Memory[V]) Get(key string) (value V, exists bool) {
	index := s.Index(key, s.count)
	tmp := s.readMemory(index)
	s.readMemoryNode(index, func() { value, exists = tmp[key] })
	return
}

func (s *Memory[V]) Exists(key string) (exists bool) {
	index := s.Index(key, s.count)
	tmp := s.readMemory(index)
	s.readMemoryNode(index, func() { _, exists = tmp[key] })
	return
}

func (s *Memory[V]) Set(key string, value V) {
	index := s.Index(key, s.count)
	tmp := s.readMemory(index)
	s.writeMemoryNode(index, func() { tmp[key] = value })
}

func (s *Memory[V]) Del(key string) {
	index := s.Index(key, s.count)
	tmp := s.readMemory(index)
	s.writeMemoryNode(index, func() { delete(tmp, key) })
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
			tmp := s.readMemory(index)
			s.readMemoryNode(index, func() {
				tmpMap := make(map[string]V, len(tmp))
				for k, v := range tmp {
					tmpMap[k] = v
				}
				lists <- &node{
					index: index,
					value: tmpMap,
				}
			})
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
			s.writeMemory(index, make(map[string]V, memoryMapCapacity))
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
			tmp := s.readMemory(index)
			s.readMemoryNode(index, func() {
				for k, v := range tmp {
					fc(index, k, v)
				}
			})
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
			tmp := s.readMemory(index)
			memory := make(map[string]V, memoryMapCapacity)
			s.writeMemoryNode(index, func() {
				for k, v := range tmp {
					memory[k] = v
				}
			})
			s.writeMemory(index, memory)
		}(i)
	}
	wg.Wait()
}
