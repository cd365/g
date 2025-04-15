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

// GoPool Implement a coroutine pool so that it can process tasks concurrently.
type GoPool struct {
	// poolCapacity Coroutine pool capacity.
	poolCapacity uint32

	// status Coroutine pool status.
	status int32

	// running The number of coroutines running.
	running int32

	// bufferTasks Task pool buffer size.
	bufferTasks uint32

	// ctx Basic context.
	ctx context.Context

	// poolCtx Coroutine pool context.
	poolCtx context.Context

	// poolCancel Coroutine pool cancelFunc.
	poolCancel context.CancelFunc

	// shutdown Make sure the stop logic is only executed once.
	shutdown *sync.Once

	// wg sync.WaitGroup.
	wg *sync.WaitGroup

	// tasks Task Channel.
	tasks chan func()

	// clean Customize your to-do task list, If not set, all pending tasks will be discarded.
	clean func(task func())

	// mutexTasks Mutex.
	mutexTasks *Mutex
}

// NewGoPool Create a coroutine pool object.
func NewGoPool(ctx context.Context, bufferTasks uint32) *GoPool {
	cpus := uint32(runtime.NumCPU())
	if bufferTasks < 1 {
		bufferTasks = cpus * 10
	}
	if ctx == nil {
		ctx = context.Background()
	}
	pool := &GoPool{
		poolCapacity: cpus,
		bufferTasks:  bufferTasks,
		ctx:          ctx,
		mutexTasks:   NewMutex(),
	}
	return pool.init()
}

// init Initialize the coroutine pool.
func (s *GoPool) init() *GoPool {
	poolCtx, poolCancel := context.WithCancel(s.ctx)
	s.poolCtx, s.poolCancel = poolCtx, poolCancel
	s.shutdown = &sync.Once{}
	s.wg = &sync.WaitGroup{}
	s.tasks = make(chan func(), s.bufferTasks)
	s.status = GoPoolStatusNotStarted
	return s
}

// GetPoolCapacity Get pool capacity.
func (s *GoPool) GetPoolCapacity() uint32 {
	return s.poolCapacity
}

// SetPoolCapacity Set pool capacity.
func (s *GoPool) SetPoolCapacity(poolCapacity uint32) *GoPool {
	if poolCapacity == 0 || poolCapacity > 1<<16 {
		return s
	}
	s.poolCapacity = poolCapacity
	return s
}

// Start Begin running the coroutine pool.
func (s *GoPool) Start() *GoPool {
	if atomic.LoadInt32(&s.status) == GoPoolStatusStopped {
		atomic.StoreInt32(&s.status, GoPoolStatusNotStarted)
		s.init()
	}
	if !atomic.CompareAndSwapInt32(&s.status, GoPoolStatusNotStarted, GoPoolStatusRunning) {
		return s
	}
	for i := uint32(0); i < s.poolCapacity; i++ {
		s.wg.Add(1)
		go s.work()
	}
	return s
}

func (s *GoPool) work() {
	defer s.wg.Done()
	defer atomic.AddInt32(&s.running, -1)

	atomic.AddInt32(&s.running, 1)
	for {
		select {
		case <-s.poolCtx.Done():
			return
		case task, ok := <-s.tasks:
			if !ok {
				return
			}
			if task != nil {
				task()
			}
		}
	}
}

// Submit adds a task to the coroutine pool.
// Returns true if the task is successfully queued, false if the task is nil or the pool is stopping/stopped.
func (s *GoPool) Submit(task func()) bool {
	if task == nil {
		return false
	}
	if status := atomic.LoadInt32(&s.status); status == GoPoolStatusStopping || status == GoPoolStatusStopped {
		return false
	}
	ok := false
	s.mutexTasks.WithLock(func() {
		if s.tasks != nil {
			select {
			case s.tasks <- task:
				ok = true
			case <-s.poolCtx.Done():
			}
		}
	})
	return ok
}

// Pending Get the number of pending tasks.
func (s *GoPool) Pending() int {
	return len(s.tasks)
}

// Clean Customize your to-do task list, If not set, all pending tasks will be discarded.
func (s *GoPool) Clean(clean func(work func())) *GoPool {
	if clean != nil {
		s.clean = clean
	}
	return s
}

// cleanTasks processes remaining tasks in the queue using the custom clean function, if set.
func (s *GoPool) cleanTasks(ctx context.Context) {
	if clean := s.clean; clean != nil {
		for {
			select {
			case <-ctx.Done():
				return
			case task, ok := <-s.tasks:
				if !ok {
					return
				}
				if task != nil {
					clean(task)
				}
			}
		}
	}
}

// Stop Stop the coroutine pool.
func (s *GoPool) Stop(ctx context.Context) *GoPool {
	s.shutdown.Do(func() {
		if atomic.CompareAndSwapInt32(&s.status, GoPoolStatusRunning, GoPoolStatusStopping) {
			s.poolCancel()

			if ctx == nil {
				ctx = context.Background()
			}
			s.mutexTasks.WithLock(func() { close(s.tasks); s.cleanTasks(ctx); s.tasks = nil })

			atomic.StoreInt32(&s.status, GoPoolStatusStopped)
		}
	})
	return s
}

// Wait Wait for all coroutines to exit.
func (s *GoPool) Wait() *GoPool {
	s.wg.Wait()
	return s
}

// Status Get the current coroutine pool status.
func (s *GoPool) Status() int32 {
	return atomic.LoadInt32(&s.status)
}

// Running Get the number of coroutines currently running in the coroutine pool.
func (s *GoPool) Running() int32 {
	return atomic.LoadInt32(&s.running)
}
