package g

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNewGoPool(t *testing.T) {
	fmt.Println(1 << 16)
	p := NewGoPool(nil, 1<<16).
		Start()

	// When the coroutine pool exits, customize the to-do task list, If not set, all pending tasks will be discarded.
	p.Clean(func(work func()) { work() })

	all := int32(0)

	{
		wg := sync.WaitGroup{}

		for i := 0; i < 500; i++ {
			wg.Add(1)
			go func(a int) {
				defer wg.Done()
				for j := 0; j < 10000; j++ {
					p.Submit(func() {
						// fmt.Println("first", a)
						atomic.AddInt32(&all, 1)
					})
				}
			}(i)
		}
		wg.Wait()
		p.Stop(nil).Wait()
	}

	{
		wg := sync.WaitGroup{}
		p.Start()
		for i := 0; i < 500; i++ {
			wg.Add(1)
			go func(a int) {
				defer wg.Done()
				for j := 0; j < 10000; j++ {
					p.Submit(func() {
						// fmt.Println("twice", a)
						atomic.AddInt32(&all, 1)
					})
				}
			}(i)
		}
		wg.Wait()
		p.Stop(nil).Wait()
	}

	fmt.Println(all)
}
