package main

import (
	"net"
	"sync"
)

type ConnKeeper struct {
	sync.Mutex
	connections map[string]net.Conn
}

type IService interface {
	Start()
}
