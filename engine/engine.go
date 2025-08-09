package engine

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/InWILL/MioSocks/windivert"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/metacubex/mihomo/constant"
)

const (
	Filter1 = "outbound and !loopback and !ipv6 and (tcp) and event == CONNECT"
	Filter2 = "outbound and !loopback and !ipv6 and (tcp)"
	mtu     = 1500
)

type Packet struct {
	buffer    []byte
	address   *windivert.Address
	timestamp time.Time
	tuple     Tuple
}

type Tuple struct {
	Protocol uint8
	SrcPort  uint16
	DstPort  uint16
}

type Engine struct {
	hSocket  *windivert.Handle
	hNetwork *windivert.Handle
	channel  chan Packet
	Process  map[uint32]bool
	session  sync.Map
	writer   io.Writer
	queue    Queue[Packet]
	dialer   constant.Proxy
}

func NewEngine(dialer constant.Proxy) *Engine {
	h1, err := windivert.Open(Filter1, windivert.LayerSocket, 0, windivert.FlagRecvOnly|windivert.FlagSniff)
	if err != nil {
		panic(err)
	}

	h2, err := windivert.Open(Filter2, windivert.LayerNetwork, 1, 0)
	if err != nil {
		panic(err)
	}

	engine := &Engine{
		hSocket:  h1,
		hNetwork: h2,
		channel:  make(chan Packet),
		Process:  make(map[uint32]bool),
		dialer:   dialer,
	}
	engine.writer = engine.NewStack()
	return engine
}

func (e *Engine) Start() {
	go e.SocketLayer()
	go e.NetworkLayer()
	go e.PacketHandler()
}

func (e *Engine) SocketLayer() {
	buffer := make([]byte, mtu)
	address := &windivert.Address{}
	for {
		_, err := e.hSocket.Recv(buffer, address)
		if err != nil {
			log.Printf("[SocketLayer] Failed to recv packet: %v\n", err)
		}

		Protocol := address.Socket().Protocol
		SrcPort := address.Socket().LocalPort
		DstPort := address.Socket().RemotePort
		PID := address.Socket().ProcessID

		if val, ok := e.Process[PID]; ok {
			tuple := Tuple{
				Protocol: Protocol,
				SrcPort:  SrcPort,
				DstPort:  DstPort,
			}
			if val {
				e.session.LoadOrStore(tuple, true)
			} else {
				e.session.LoadOrStore(tuple, false)
			}
		} else {
			name, _ := GetProcName(PID)
			log.Printf("Program:%s PID:%d %d:%d\n", name, PID, SrcPort, DstPort)
			if name == "MapleStory.exe" {
				e.Process[PID] = true
			} else {
				e.Process[PID] = false
			}
		}

	}
}

func (e *Engine) NetworkLayer() {
	buffer := make([]byte, mtu)
	address := &windivert.Address{}
	for {
		_, err := e.hNetwork.Recv(buffer, address)
		if err != nil {
			log.Printf("[NetworkLayer] Failed to recv packet: %v\n", err)
		}

		packet := gopacket.NewPacket(buffer, layers.LayerTypeIPv4, gopacket.Default)
		ip_header := packet.Layer(layers.LayerTypeIPv4)
		tcp_header := packet.Layer(layers.LayerTypeTCP)
		if ip_header != nil && tcp_header != nil {
			ip, _ := ip_header.(*layers.IPv4)
			tcp, _ := tcp_header.(*layers.TCP)

			tuple := Tuple{
				Protocol: uint8(ip.Protocol),
				SrcPort:  uint16(tcp.SrcPort),
				DstPort:  uint16(tcp.DstPort),
			}
			if val, ok := e.session.Load(tuple); ok {
				if val == true {
					e.writer.Write(buffer)
				} else {
					_, err = e.hNetwork.Send(buffer, address)
					if err != nil {
						log.Printf("[NetworkLayer] Failed to send packet: %v\n", err)
					}
				}
			} else {
				packet := Packet{
					buffer:    buffer,
					address:   address,
					timestamp: time.Now(),
					tuple:     tuple,
				}
				e.channel <- packet
			}
		}
	}
}

func (e *Engine) PacketHandler() {
	timeout := 5 * time.Millisecond
	ticker := time.NewTicker(timeout)
	for {
		select {
		case packet := <-e.channel:
			e.queue.push(packet)
		case <-ticker.C:
			now := time.Now()
			for !e.queue.empty() {
				item := e.queue.front()
				if now.Before(item.timestamp.Add(timeout)) {
					break
				}
				e.queue.pop()
				if val, ok := e.session.Load(item.tuple); ok && (val == true) {
					e.writer.Write(item.buffer)
				} else {
					_, err := e.hNetwork.Send(item.buffer, item.address)
					if err != nil {
						log.Printf("[PacketHandler] Failed to send packet: %v\n", err)
					}
				}
			}
		}
	}
}

func (e *Engine) NetStack_Output(buffer []byte) (int, error) {
	log.Println("NetStack_Output")
	address := windivert.Address{}
	const f = uint8(0x01<<0) | uint8(0x01<<1) | uint8(0x01<<2)
	address.Flags |= f
	//address.Network().InterfaceIndex = 7
	//windivert.HelperCalcChecksums(buffer, &address)
	size, err := e.hNetwork.Send(buffer, &address)
	return size, err
}
