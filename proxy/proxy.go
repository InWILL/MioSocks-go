package proxy

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"log"
	"net"
	"net/netip"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/constant"
)

type Proxy struct {
	constant.Proxy
}

func NewProxy(mapping map[string]any) *Proxy {
	proxy, err := adapter.ParseProxy(mapping)
	if err != nil {
		log.Fatalf("Failed to parse proxy: %v", err)
	}
	return &Proxy{
		Proxy: proxy,
	}
}

func (p *Proxy) HandleConnection(conn net.Conn) {
	bufReader := bufio.NewReader(conn)
	peek, err := bufReader.Peek(1)
	if err != nil {
		log.Printf("[SOCKS5] Failed to read from connection: %v", err)
		conn.Close()
		return
	}

	switch peek[0] {
	case 0x05:
		p.HandleSocks5(bufReader, conn)
	default:
		// Config.handleRestAPI(bufReader, conn)
	}
}

func (p *Proxy) HandleSocks5(r *bufio.Reader, conn net.Conn) {
	log.Println("[SOCKS5] New connection established")

	buf := make([]byte, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		log.Println("[SOCKS5] handshake failed:", err)
		return
	}
	nmethods := buf[1]
	if _, err := io.ReadFull(r, make([]byte, nmethods)); err != nil {
		log.Println("[SOCKS5] methods read failed:", err)
		return
	}

	// No Auth reply
	conn.Write([]byte{0x05, 0x00})

	// 3. 读取请求头
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		log.Println("[SOCKS5] Request header error:", err)
		return
	}
	cmd := header[1]
	atyp := header[3]
	if cmd != 0x01 {
		log.Println("[SOCKS5] Only CONNECT supported")
		conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // command not supported
		return
	}

	// 4. 解析地址
	metadata := &constant.Metadata{
		NetWork: constant.TCP,
	}
	switch atyp {
	case 0x01: // IPv4
		addr := make([]byte, 6)
		if _, err := io.ReadFull(r, addr); err != nil {
			log.Println("[SOCKS5] IPv4 address read error:", err)
			return
		}
		ip := net.IP(addr[:4])
		metadata.DstIP = netip.MustParseAddr(ip.String())
		metadata.DstPort = binary.BigEndian.Uint16(addr[4:])
		log.Printf("[%s] Connecting to: %s:%d\n", p.Type(), metadata.DstIP, metadata.DstPort)

	case 0x03: // Domain
		lenByte, err := r.ReadByte()
		if err != nil {
			log.Println("[SOCKS5] Domain length read error:", err)
			return
		}
		domain := make([]byte, lenByte+2)
		if _, err := io.ReadFull(r, domain); err != nil {
			log.Println("[SOCKS5] Domain read error:", err)
			return
		}
		metadata.Host = string(domain[:lenByte])
		metadata.DstPort = binary.BigEndian.Uint16(domain[lenByte:])
		log.Printf("[%s] Connecting to: %s:%d\n", p.Type(), metadata.Host, metadata.DstPort)

	default:
		log.Println("[SOCKS5] Unsupported address type")
		conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // address type not supported
		return
	}

	// Dial through the proxy
	ctx := context.Background()
	dstConn, err := p.DialContext(ctx, metadata)
	if err != nil {
		log.Printf("[%s] Dial error: %v", p.Type(), err)
		conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // connection refused
		return
	}

	// 6. 发送成功响应
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// 7. 数据转发
	go forward(dstConn, conn)
	go forward(conn, dstConn)
}

func forward(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}
