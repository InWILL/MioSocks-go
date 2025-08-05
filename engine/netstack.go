package engine

import (
	"io"
	"log"
	"net"

	"github.com/eycorsican/go-tun2socks/core"
)

type tcpHandler struct {
}

func NewStack(fn func([]byte) (int, error)) io.Writer {
	core.RegisterTCPConnHandler(NewTCPHandler())
	core.RegisterOutputFn(fn)
	netstack := core.NewLWIPStack()
	return netstack
}

func NewTCPHandler() core.TCPConnHandler {
	return &tcpHandler{}
}

func (h *tcpHandler) Handle(conn net.Conn, target *net.TCPAddr) error {
	log.Printf("%s => %s:%d\n", conn.LocalAddr().String(), target.IP, target.Port)
	return nil
}
