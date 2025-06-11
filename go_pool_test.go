package g

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewGoPool(t *testing.T) {
	test1()

	test2()

	test3()
}

func test1() {
	p := NewGoPool(1 << 16)

	// When the coroutine pool exits, customize the to-do task list, If not set, all pending tasks will be discarded.
	p.Clean(func(t GoPoolTasker) {
		// Currently in the cleanup task, maybe there is no need to execute the original task logic.
		t.Execute(t) // It may need to be commented out.

		// Implement your own custom cleanup logic. TODO...
		fmt.Printf("test1 call by clean: %p %v\n", t, t.Load())
	})

	all := int32(0)

	{
		wg := sync.WaitGroup{}
		p.Start()
		for i := 0; i < 500; i++ {
			wg.Add(1)
			go func(a int) {
				defer wg.Done()
				for j := 0; j < 1000; j++ {
					newTask := NewGoPoolTasker(func(t GoPoolTasker) {
						defer atomic.AddInt32(&all, 1)
						// TODO your logic
					})
					p.Submit(newTask)
				}
			}(i)
		}
		wg.Wait()
		p.Stop()
	}

	// Secondary startup.
	{
		wg := sync.WaitGroup{}
		p.Start()
		for i := 0; i < 500; i++ {
			wg.Add(1)
			go func(a int) {
				defer wg.Done()
				for j := 0; j < 1000; j++ {
					newTask := NewGoPoolTasker(func(t GoPoolTasker) {
						defer atomic.AddInt32(&all, 1)
						// TODO your logic
					})
					p.Submit(newTask)
				}
			}(i)
		}
		wg.Wait()
		p.Stop()
	}

	fmt.Println("test1", all)
}

func test2() {
	input := int32(0)
	done := int32(0)
	success := int32(0)
	count := int32(5000000)
	stopCount := count / 10 * 7

	p := NewGoPool(1 << 16)
	p.Clean(func(t GoPoolTasker) {
		// Currently in the cleanup task, maybe there is no need to execute the original task logic.
		t.Execute(t) // It may need to be commented out.

		// Implement your own custom cleanup logic. TODO...
		fmt.Printf("test2 call by clean: %p %v\n", t, t.Load())
	})
	p.Start()

	go func() {
		for i := int32(0); i < count; i++ {
			newTask := NewGoPoolTasker(func(t GoPoolTasker) {
				defer atomic.AddInt32(&done, 1)

				// custom logic
				if v := t.Load(); v != nil {
					if iv, ok := v.(int32); ok {
						if iv&1 == 1 {
							return
						}
					}
				}

				atomic.AddInt32(&success, 1)
			})
			newTask.Store(i)
			if p.Submit(newTask) {
				atomic.AddInt32(&input, 1)
			}
		}
	}()

	// In some cases, you may need to stop the GoPool.
	go func() {
		for {
			if atomic.LoadInt32(&input) >= stopCount {
				p.Stop()
				fmt.Println("test2", "stop called")
				break
			}
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if p.Status() != GoPoolStatusRunning {
			break
		}
		<-ticker.C
		fmt.Println("test2", "pending:", p.Pending(), "input:", atomic.LoadInt32(&input), "done:", atomic.LoadInt32(&done), "success:", atomic.LoadInt32(&success))
	}
}

func test3() {
	p := NewGoPool(1 << 16)
	p.Clean(func(t GoPoolTasker) {
		// Currently in the cleanup task, maybe there is no need to execute the original task logic.
		// t.Execute(t) // It may need to be commented out.

		// Implement your own custom cleanup logic. TODO...
		fmt.Printf("test3 call by clean: %p %v\n", t, t.Load())
	})
	p.Start()

	count := 10000000
	added := int32(0)
	called := int32(0)
	for a := 0; a < 10; a++ {
		go func() {
			total := count / 10
			for i := 0; i < total; i++ {
				if p.Submit(NewGoPoolTasker(func(t GoPoolTasker) {
					atomic.AddInt32(&called, 1)
				})) {
					atomic.AddInt32(&added, 1)
				}
			}
		}()
	}

	time.Sleep(time.Second * 3)
	p.Stop()
	fmt.Println("test3", "count:", count, "called:", atomic.LoadInt32(&called), "add:", atomic.LoadInt32(&added))
}
