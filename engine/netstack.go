package engine

import (
	"context"
	"io"
	"log"
	"net"
	"net/netip"

	"github.com/eycorsican/go-tun2socks/core"
	"github.com/metacubex/mihomo/constant"
)

type tcpHandler struct {
	constant.Proxy
}

func (e *Engine) NewStack() io.Writer {
	core.RegisterTCPConnHandler(NewTCPHandler(e.dialer))
	core.RegisterOutputFn(e.NetStack_Output)
	netstack := core.NewLWIPStack()
	return netstack
}

func NewTCPHandler(dialer constant.Proxy) core.TCPConnHandler {
	return &tcpHandler{
		Proxy: dialer,
	}
}

func (h *tcpHandler) Handle(conn net.Conn, target *net.TCPAddr) error {
	log.Printf("%s => %s:%d\n", conn.LocalAddr().String(), target.IP, target.Port)
	metadata := &constant.Metadata{
		NetWork: constant.TCP,
	}
	metadata.DstIP = netip.MustParseAddr(target.IP.String())
	metadata.DstPort = uint16(target.Port)

	ctx := context.Background()
	dstConn, err := h.DialContext(ctx, metadata)
	if err != nil {
		return err
	}

	go forward(dstConn, conn)
	go forward(conn, dstConn)

	return nil
}

func forward(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}
