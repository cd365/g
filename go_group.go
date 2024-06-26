package g

import (
	"bytes"
	"fmt"
	"runtime/debug"
	"sync"
)

type GoGroup struct {
	Go func(fc func())

	// Stack Custom handling panicked debug.Stack(), nil value will not call the receiver.
	Stack func(stack []byte)

	// shutdownOnce Make sure to write a shutdown signal to `shutdownWait` only once.
	shutdownOnce *sync.Once

	// ReadShutdown Read the shutdown error.
	ReadShutdown chan error

	// Shutdown Unconditionally exit the current coroutine group.
	Shutdown func(err error)

	// ShutdownError Exit the current coroutine group when an error occurs.
	ShutdownError func(err error)

	mutex *sync.Mutex

	// exit Last in first out, when the function list is called, all coroutines are exited.
	exit []func()

	// quit Last in first out, when the function list is called, not all coroutines have exited.
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

func NewGoGroup() *GoGroup {
	s := &GoGroup{
		wg:           &sync.WaitGroup{},
		mutex:        &sync.Mutex{},
		shutdownOnce: &sync.Once{},
		ReadShutdown: make(chan error, 1),
	}

	s.Go = func(fc func()) {
		if fc == nil {
			return
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			stack := s.Stack
			if stack != nil {
				defer func() {
					if msg := recover(); msg != nil {
						buf := bytes.NewBuffer(nil)
						buf.WriteString(fmt.Sprintf("<<< %v >>>\n", msg))
						buf.Write(debug.Stack())
						stack(buf.Bytes())
					}
				}()
			}
			fc()
		}()
	}

	s.Shutdown = func(err error) {
		s.shutdownOnce.Do(func() {
			s.ReadShutdown <- err
			close(s.ReadShutdown)
		})
	}
	s.ShutdownError = func(err error) {
		if err != nil {
			s.Shutdown(err)
		}
	}

	return s
}
