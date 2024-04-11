package g

import (
	"context"
	"net"
	"net/rpc"
	"testing"
	"time"
)

type RpcDemo struct{}

type RpcDemoArgs struct {
	A, B int
}

type RpcDemoResp struct {
	C int
}

func (t *RpcDemo) Add(args RpcDemoArgs, reply *RpcDemoResp) error {
	reply.C = args.A + args.B
	return nil
}

func (t *RpcDemo) Mul(args *RpcDemoArgs, reply *RpcDemoResp) error {
	reply.C = args.A * args.B
	return nil
}

func TestNewRpcServer(t *testing.T) {
	listener, err := net.Listen("tcp", ":7890")
	if err != nil {
		t.Log(err.Error())
		return
	}
	server := NewJsonRpcServer(listener, func(server *rpc.Server) error {
		return server.RegisterName("demo", &RpcDemo{})
	})
	// wait service goroutine exit
	defer func() { <-time.After(time.Millisecond * 100) }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start service
	go func() { _ = server.Start(ctx) }()

	// connect to service
	conn, err := net.Dial("tcp", "localhost:7890")
	if err != nil {
		t.Log(err.Error())
		return
	}

	cli := NewJsonRpcClient(conn)
	if err != nil {
		t.Log(err.Error())
		return
	}
	defer func() { _ = cli.Close() }()

	// direct call
	req := &RpcDemoArgs{
		A: 1,
		B: 2,
	}
	res := &RpcDemoResp{}
	err = cli.Call("demo.Add", req, res)
	if err != nil {
		t.Log(err.Error())
		return
	}
	t.Log("rpc call success, result is:", res.C)

	// asynchronous call
	if time.Now().Unix()%2 == 0 {
		done := make(chan *rpc.Call, 1)
		if err = cli.Go("demo.Mul", req, res, done).Error; err != nil {
			t.Log(err.Error())
			return
		}
		<-done
	} else {
		call := cli.Go("demo.Mul", req, res, nil)
		if call.Error != nil {
			t.Log(call.Error.Error())
			return
		}
		<-call.Done
	}
	t.Log("rpc call success, result is:", res.C)
}
