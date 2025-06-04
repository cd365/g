package g

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNewGoPool(t *testing.T) {
	p := NewGoPool(nil, 1<<16)

	// When the coroutine pool exits, customize the to-do task list, If not set, all pending tasks will be discarded.
	p.Clean(func(work func()) { work() })

	all := int32(0)

	{
		wg := sync.WaitGroup{}
		p.Start()
		for i := 0; i < 500; i++ {
			wg.Add(1)
			go func(a int) {
				defer wg.Done()
				for j := 0; j < 1000; j++ {
					p.Submit(func() { atomic.AddInt32(&all, 1) })
				}
			}(i)
		}
		wg.Wait()
		p.Stop().Wait()
	}

	{
		wg := sync.WaitGroup{}
		p.Start()
		for i := 0; i < 500; i++ {
			wg.Add(1)
			go func(a int) {
				defer wg.Done()
				for j := 0; j < 1000; j++ {
					p.Submit(func() { atomic.AddInt32(&all, 1) })
				}
			}(i)
		}
		wg.Wait()
		p.Stop().Wait()
	}

	fmt.Println(all)
}
