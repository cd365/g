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

// GoTask Package custom task and support storage of custom data.
type GoTask interface {
	// Load Loading stored custom data.
	Load() (any, func(i any))

	// Store Storing custom data.
	Store(i any, execute func(i any))

	// Execute Calling custom task function.
	Execute()
}

type goTask struct {
	storage any
	execute func(i any)
}

func (s *goTask) Load() (any, func(i any)) {
	return s.storage, s.execute
}

func (s *goTask) Store(i any, execute func(i any)) {
	if execute != nil {
		s.storage, s.execute = i, execute
	}
}

func (s *goTask) Execute() {
	storage, execute := s.Load()
	if execute != nil {
		execute(storage)
	}
}

func NewGoTask() GoTask {
	return &goTask{}
}

// GoPool Implement a coroutine pool so that it can process tasks concurrently.
type GoPool struct {
	// ctx Coroutine pool context.
	ctx context.Context

	// cancel Coroutine pool cancelFunc.
	cancel context.CancelFunc

	// wg sync.WaitGroup.
	wg *sync.WaitGroup

	// tasks Task Channel.
	tasks chan GoTask

	// clean Customize your to-do task list, If not set, all pending tasks will be discarded.
	clean func(task GoTask)

	// mutex Task Mutex.
	mutex *Mutex

	// goNumbers Number of coroutines.
	goNumbers uint32

	// taskQueueCapacity Task queue capacity.
	taskQueueCapacity uint32

	// status Coroutine pool status.
	status int32

	// running The number of coroutines running.
	running int32
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
		mutex:             NewMutex(),
	}
	return pool.init()
}

// init Initialize the coroutine pool.
func (s *GoPool) init() *GoPool {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.wg = &sync.WaitGroup{}
	s.tasks = make(chan GoTask, s.taskQueueCapacity)
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
func (s *GoPool) Clean(clean func(t GoTask)) *GoPool {
	s.clean = clean
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
	defer func() {
		s.wg.Done()
		atomic.AddInt32(&s.running, -1)
	}()
	atomic.AddInt32(&s.running, 1)
	for atomic.LoadInt32(&s.status) == GoPoolStatusRunning {
		select {
		case <-s.ctx.Done():
			return
		case tmp, ok := <-s.tasks:
			if ok && tmp != nil {
				tmp.Execute()
			}
		}
	}
}

// Submit adds a task to the coroutine pool.
// Returns true if the task is successfully queued, false if the task is nil or the pool is stopping/stopped.
func (s *GoPool) Submit(task GoTask) bool {
	if task == nil || atomic.LoadInt32(&s.status) != GoPoolStatusRunning {
		return false
	}
	result := false
	s.mutex.WithLock(func() {
		if atomic.LoadInt32(&s.status) == GoPoolStatusRunning {
			select {
			case <-s.ctx.Done():
			case s.tasks <- task:
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
		s.mutex.WithLock(func() {
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
