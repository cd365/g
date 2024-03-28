package g

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestNewGoGroup(t *testing.T) {
	// ...

	c := NewGoGroup()
	defer c.Exit()
	defer c.Wait()
	defer c.Quit()
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// ...

	c.Go(func() {
		<-time.After(time.Second * 5)
		c.Shutdown(errors.New("subjective withdrawal from the current program"))
	})
	c.Go(func() {
		<-time.After(time.Second)
		panic("test panic recover")
	})

	// ...

	// c.Go(func() {
	// 	notify := make(chan os.Signal, 1)
	// 	signal.Notify(notify, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	// 	select {
	// 	case <-ctx.Done():
	// 	case sig := <-notify:
	// 		c.Shutdown(errors.New(sig.String()))
	// 	}
	// })

	if err := c.ShutdownWait(); err != nil {
		fmt.Println("the program is about to exit -->", err.Error())
	}

	// ...
}
