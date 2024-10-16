package g

import (
	"bytes"
	"fmt"
	"runtime/debug"
	"sync"
)

type GoGroup struct {
	// coroutine Start a coroutine.
	coroutine func(fc func())

	// stack Custom handling panicked debug.Stack(), nil value will not call the receiver.
	stack func(stack []byte)

	// exit Stop the current coroutine group.
	exit func(err error)

	// exitErr Read the error of Exit().
	exitErr chan error

	// doneOnce Make sure to write a shutdown signal to `shutdownWait` only once.
	exitOnce *sync.Once

	// listExit Last in first out, when the function list is called, all coroutines are exited.
	listExit []func()

	// listQuit Last in first out, when the function list is called, not all coroutines have exited.
	listQuit []func()

	mutex *sync.Mutex

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

func (s *GoGroup) Stack(stack func(stack []byte)) {
	s.stack = stack
}

func (s *GoGroup) PushExit(fc func()) {
	if fc != nil {
		s.withLock(func() { s.listExit = append(s.listExit, fc) })
	}
}

func (s *GoGroup) CallExit() {
	for i := len(s.listExit) - 1; i >= 0; i-- {
		s.listExit[i]()
	}
}

func (s *GoGroup) PushQuit(fc func()) {
	if fc != nil {
		s.withLock(func() { s.listQuit = append(s.listQuit, fc) })
	}
}

func (s *GoGroup) CallQuit() {
	for i := len(s.listQuit) - 1; i >= 0; i-- {
		s.listQuit[i]()
	}
}

func (s *GoGroup) Go(fc func()) {
	s.coroutine(fc)
}

func (s *GoGroup) Exit(err error) {
	s.exit(err)
}

func (s *GoGroup) ExitErr() <-chan error {
	return s.exitErr
}

func (s *GoGroup) Wait() {
	s.wg.Wait()
}

func NewGoGroup() *GoGroup {
	s := &GoGroup{
		exitErr:  make(chan error, 1),
		exitOnce: &sync.Once{},
		mutex:    &sync.Mutex{},
		wg:       &sync.WaitGroup{},
	}

	s.coroutine = func(fc func()) {
		if fc == nil {
			return
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			stack := s.stack
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

	s.exit = func(err error) {
		s.exitOnce.Do(func() {
			s.exitErr <- err
			close(s.exitErr)
		})
	}

	return s
}
