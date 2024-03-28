package g

import (
	"bytes"
	"fmt"
	"runtime/debug"
	"sync"
)

type GoGroup struct {
	Go func(fc func())

	Stack func(stack []byte)

	shutdownOnce *sync.Once

	shutdownWait chan error

	Shutdown func(err error)

	mutex *sync.Mutex

	// last in first out, when the function list is called, all coroutines are exited.
	exit []func()

	// last in first out, when the function list is called, not all coroutines have exited.
	quit []func()

	wg *sync.WaitGroup
}

func (s *GoGroup) withLock(fc func()) {
	if fc == nil {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	fc()
}

func (s *GoGroup) AddExit(fc func()) {
	if fc != nil {
		s.withLock(func() { s.exit = append(s.exit, fc) })
	}
}

func (s *GoGroup) Exit() {
	for i := len(s.exit) - 1; i >= 0; i-- {
		s.exit[i]()
	}
}

func (s *GoGroup) AddQuit(fc func()) {
	if fc != nil {
		s.withLock(func() { s.quit = append(s.quit, fc) })
	}
}

func (s *GoGroup) Quit() {
	for i := len(s.quit) - 1; i >= 0; i-- {
		s.quit[i]()
	}
}

func (s *GoGroup) Wait() {
	s.wg.Wait()
}

func (s *GoGroup) ShutdownWait() error {
	return <-s.shutdownWait
}

func NewGoGroup() *GoGroup {
	s := &GoGroup{
		wg:           &sync.WaitGroup{},
		mutex:        &sync.Mutex{},
		shutdownOnce: &sync.Once{},
		shutdownWait: make(chan error, 1),
	}

	s.Go = func(fc func()) {
		if fc == nil {
			return
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() {
				if msg := recover(); msg != nil {
					buf := bytes.NewBuffer(nil)
					buf.WriteString(fmt.Sprintf("<<< %v >>>\n", msg))
					buf.Write(debug.Stack())
					if s.Stack != nil {
						s.Stack(buf.Bytes())
					} else {
						fmt.Println(buf.String())
					}
				}
			}()
			fc()
		}()
	}

	s.Shutdown = func(err error) {
		s.shutdownOnce.Do(func() {
			s.shutdownWait <- err
			close(s.shutdownWait)
		})
	}

	return s
}
