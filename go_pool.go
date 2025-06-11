package g

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
)

const (
	GoPoolStatusNotStarted int32 = iota // Not started
	GoPoolStatusRunning                 // Running
	GoPoolStatusStopping                // Stopping
	GoPoolStatusStopped                 // Stopped
)

// GoPoolTasker Package custom task and support storage of custom data.
type GoPoolTasker interface {
	// Load Loading stored custom data.
	Load() interface{}

	// Store Storing custom data.
	Store(i interface{}) GoPoolTasker

	// Execute Calling custom task function.
	Execute(t GoPoolTasker) GoPoolTasker
}

type goPoolTasker struct {
	callStatus int32
	execute    func(t GoPoolTasker)
	storage    interface{}
}

func (s *goPoolTasker) Load() interface{} {
	return s.storage
}

func (s *goPoolTasker) Store(i interface{}) GoPoolTasker {
	s.storage = i
	return s
}

func (s *goPoolTasker) Execute(t GoPoolTasker) GoPoolTasker {
	if atomic.CompareAndSwapInt32(&s.callStatus, 0, 1) && s.execute != nil {
		s.execute(t)
	}
	return s
}

func NewGoPoolTasker(fc func(t GoPoolTasker)) GoPoolTasker {
	return &goPoolTasker{
		execute: fc,
	}
}

// GoPool Implement a coroutine pool so that it can process tasks concurrently.
type GoPool struct {
	// goNumbers Number of coroutines.
	goNumbers uint32

	// taskQueueCapacity Task queue capacity.
	taskQueueCapacity uint32

	// status Coroutine pool status.
	status int32

	// running The number of coroutines running.
	running int32

	// ctx Coroutine pool context.
	ctx context.Context

	// cancel Coroutine pool cancelFunc.
	cancel context.CancelFunc

	// wg sync.WaitGroup.
	wg *sync.WaitGroup

	// tasks Task Channel.
	tasks chan GoPoolTasker

	// clean Customize your to-do task list, If not set, all pending tasks will be discarded.
	clean func(tasker GoPoolTasker)

	// mutexTasks Mutex.
	mutexTasks *Mutex
}

// NewGoPool Create a coroutine pool object.
func NewGoPool(taskQueueCapacity uint32) *GoPool {
	goNumbers := uint32(runtime.NumCPU())
	if taskQueueCapacity < 1 {
		taskQueueCapacity = goNumbers * 10
	}
	pool := &GoPool{
		goNumbers:         goNumbers,
		taskQueueCapacity: taskQueueCapacity,
		mutexTasks:        NewMutex(),
	}
	return pool.init()
}

// init Initialize the coroutine pool.
func (s *GoPool) init() *GoPool {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.wg = &sync.WaitGroup{}
	s.tasks = make(chan GoPoolTasker, s.taskQueueCapacity)
	atomic.StoreInt32(&s.status, GoPoolStatusNotStarted)
	return s
}

// GetGoNumbers Get go numbers.
func (s *GoPool) GetGoNumbers() uint32 {
	return s.goNumbers
}

// SetGoNumbers Set go numbers.
func (s *GoPool) SetGoNumbers(goNumbers uint32) *GoPool {
	if goNumbers == 0 || goNumbers > 1<<16 {
		return s
	}
	s.goNumbers = goNumbers
	return s
}

// Clean Customize your to-do task list, If not set, all pending tasks will be discarded.
func (s *GoPool) Clean(handle func(t GoPoolTasker)) *GoPool {
	s.clean = handle
	return s
}

// Start Starting the coroutine pool and return whether the start operation is successful.
func (s *GoPool) Start() bool {
	if atomic.LoadInt32(&s.status) == GoPoolStatusStopped {
		s.init()
	}
	if atomic.CompareAndSwapInt32(&s.status, GoPoolStatusNotStarted, GoPoolStatusRunning) {
		for i := uint32(0); i < s.goNumbers; i++ {
			s.wg.Add(1)
			go s.work()
		}
		return true
	}
	return false
}

// work Read the task and call the task custom function.
func (s *GoPool) work() {
	defer s.wg.Done()
	defer atomic.AddInt32(&s.running, -1)
	atomic.AddInt32(&s.running, 1)
	for atomic.LoadInt32(&s.status) == GoPoolStatusRunning {
		select {
		case <-s.ctx.Done():
			return
		case tmp, ok := <-s.tasks:
			if ok && tmp != nil {
				tmp.Execute(tmp)
			}
		}
	}
}

// Submit adds a task to the coroutine pool.
// Returns true if the task is successfully queued, false if the task is nil or the pool is stopping/stopped.
func (s *GoPool) Submit(tasker GoPoolTasker) bool {
	if tasker == nil || atomic.LoadInt32(&s.status) != GoPoolStatusRunning {
		return false
	}
	result := false
	s.mutexTasks.WithLock(func() {
		if atomic.LoadInt32(&s.status) == GoPoolStatusRunning {
			select {
			case <-s.ctx.Done():
			case s.tasks <- tasker:
				result = true
			}
		}
	})
	return result
}

// cleanTasks Processes remaining tasks in the queue using the custom clean function, if set.
func (s *GoPool) cleanTasks() {
	if clean := s.clean; clean != nil {
		for {
			task, ok := <-s.tasks
			if !ok {
				return
			}
			if task != nil {
				clean(task)
			}
		}
	}
}

// Stop Stoping the coroutine pool and return whether the stop operation is successful.
func (s *GoPool) Stop() bool {
	if atomic.CompareAndSwapInt32(&s.status, GoPoolStatusRunning, GoPoolStatusStopping) {
		defer s.wg.Wait()
		s.cancel()
		s.mutexTasks.WithLock(func() {
			close(s.tasks)
			s.cleanTasks()
		})
		atomic.StoreInt32(&s.status, GoPoolStatusStopped)
		return true
	}
	return false
}

// Pending Get the number of pending tasks.
func (s *GoPool) Pending() int {
	return len(s.tasks)
}

// Status Get the current coroutine pool status.
func (s *GoPool) Status() int32 {
	return atomic.LoadInt32(&s.status)
}

// Running Get the number of coroutines currently running in the coroutine pool.
func (s *GoPool) Running() int32 {
	return atomic.LoadInt32(&s.running)
}
