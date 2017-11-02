package main

import (
	"net/rpc"
	"net"
	"net/rpc/jsonrpc"
	"os"
	"time"
	"proto"
)

func StartJsonrpcServer() {
	rpcServer := rpc.NewServer()

	listener, error := net.Listen("tcp", "0.0.0.0:5000")
	if error != nil {
		os.Exit(1)
	}

	rpcServer.Register(proto.NewJsonrpcHandler())

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			go rpcServer.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}()
}

func main() {
	StartJsonrpcServer()
	for {
		time.Sleep(1 * time.Second)
	}
}