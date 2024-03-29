package call

import (
	"context"
	"io"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

type RpcServer struct {
	listener net.Listener
	codec    func(conn io.ReadWriteCloser) rpc.ServerCodec
	register func(server *rpc.Server) error
}

func NewRpcServer(
	listener net.Listener,
	codec func(conn io.ReadWriteCloser) rpc.ServerCodec,
	register func(server *rpc.Server) error,
) *RpcServer {
	if codec == nil {
		codec = jsonrpc.NewServerCodec
	}
	return &RpcServer{
		listener: listener,
		codec:    codec,
		register: register,
	}
}

func (s *RpcServer) Start(ctx context.Context) error {
	service := rpc.NewServer()
	if s.register != nil {
		if err := s.register(service); err != nil {
			return err
		}
	}
	for {
		select {
		case <-ctx.Done():
			return s.listener.Close()
		default:
			if conn, wer := s.listener.Accept(); wer == nil {
				go service.ServeCodec(s.codec(conn))
			}
		}
	}
}

func NewRpcClient(conn net.Conn, codec func(conn io.ReadWriteCloser) rpc.ClientCodec) *rpc.Client {
	if codec == nil {
		codec = jsonrpc.NewClientCodec
	}
	return rpc.NewClientWithCodec(codec(conn))
}

func NewJsonRpcServer(listener net.Listener, register func(server *rpc.Server) error) *RpcServer {
	return NewRpcServer(listener, jsonrpc.NewServerCodec, register)
}

func NewJsonRpcClient(conn net.Conn) *rpc.Client {
	return NewRpcClient(conn, jsonrpc.NewClientCodec)
}
