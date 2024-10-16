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
	defer c.CallExit()
	defer c.Wait()
	defer c.CallQuit()
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// ...

	c.Go(func() {
		<-time.After(time.Second * 5)
		c.Exit(errors.New("subjective withdrawal from the current program"))
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
	// 		c.Exit(errors.New(sig.String()))
	// 	}
	// })

	if err := <-c.ExitErr(); err != nil {
		fmt.Println("the program is about to exit -->", err.Error())
	}

	// ...
}
