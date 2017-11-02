package main

import (
	"sync"
	"net/rpc"
	"net"
	"net/rpc/jsonrpc"
	"time"
	"proto"
	"errors"
	"runtime"
)

const (
	PCONN_STATUS_UNKNOWN = 0
	PCONN_STATUS_CONNECTED = 1
	PCONN_STATUS_ERROR = 2
	PCONN_STATUS_CLOSED = 3
)

type PersistConn struct {
	mutex sync.Mutex
	client *rpc.Client
	addr string
	connMs int
	status int
	closeCh chan byte
	errorCh chan uint64
	lastRetry int64
	retryInterval int64
	connVer uint64
}

func (pconn *PersistConn)HealthCheck() {
	for {
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <- timer.C:
		case <- pconn.closeCh:
			goto close;
		case errVer := <- pconn.errorCh:
			if pconn.status == PCONN_STATUS_CONNECTED && errVer == pconn.connVer {
				pconn.mutex.Lock()
				pconn.status = PCONN_STATUS_ERROR
				pconn.client.Close()
				pconn.mutex.Unlock()
			}
		}
		now := time.Now().UnixNano()
		if pconn.status == PCONN_STATUS_ERROR && now - pconn.lastRetry >= pconn.retryInterval {
			conn, err := net.DialTimeout("tcp", pconn.addr, time.Duration(pconn.connMs) * time.Millisecond)
			if err == nil {
				pconn.mutex.Lock()
				pconn.connVer++
				pconn.status = PCONN_STATUS_CONNECTED
				pconn.client = jsonrpc.NewClient(conn)
				pconn.mutex.Unlock()
			}
			pconn.lastRetry = now
		}
	}
close:
	pconn.mutex.Lock()
	pconn.status = PCONN_STATUS_CLOSED
	pconn.client.Close()
	pconn.mutex.Unlock()
}

func NewPConn(addr string, connMs int, retryInterval int) *PersistConn {
	pconn := PersistConn{
		status: PCONN_STATUS_UNKNOWN,
		addr: addr,
		connMs: connMs,
		closeCh: make(chan byte, 1),
		errorCh: make(chan uint64, 1),
		lastRetry: 0,
		retryInterval: int64(retryInterval) * 1000000,
		connVer: 0,
	}

	conn, err := net.DialTimeout("tcp", addr, time.Duration(connMs) * time.Millisecond)
	if err != nil {
		pconn.status = PCONN_STATUS_ERROR
	} else {
		pconn.client = jsonrpc.NewClient(conn)
		pconn.status = PCONN_STATUS_CONNECTED
	}

	go pconn.HealthCheck()
	return &pconn
}

func (pconn *PersistConn)Destroy() {
	select {
		case pconn.closeCh <- byte(1):
		default:
	}
}

func (pconn *PersistConn)Call(serviceMethod string, args interface{}, reply interface{}) error {
	pconn.mutex.Lock()
	if pconn.status != PCONN_STATUS_CONNECTED {
		pconn.mutex.Unlock()
		return errors.New("connection's gone away")
	}
	client := pconn.client
	curVer := pconn.connVer
	pconn.mutex.Unlock()

	err := client.Call(serviceMethod, args, reply)
	if err != nil {
		select {
		case pconn.errorCh <- curVer:
		default:
		}
	}
	return err
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	pconn := NewPConn("127.0.0.1:5000", 1000, 1000)

	for i := 0; i < 100; i++ {
		go func() {
			for {
				var count int
				if err := pconn.Call("JsonrpcHandler.Get", proto.NoArgs{}, &count); err == nil {
					// fmt.Println(count)
				}
			}
		}()
	}

	for {
		time.Sleep(1 * time.Second)
	}

	pconn.Destroy()
}