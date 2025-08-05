package engine

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/imgk/divert-go"
	shadow "github.com/imgk/shadow/utils"
)

const (
	Filter1 = "outbound and !loopback and !ipv6 and (tcp) and event == CONNECT"
	Filter2 = "outbound and !loopback and !ipv6 and (tcp)"
	mtu     = 1500
)

type Packet struct {
	buffer    []byte
	address   *divert.Address
	timestamp time.Time
	tuple     Tuple
}

type Tuple struct {
	Protocol uint8
	SrcPort  uint16
	DstPort  uint16
}

type Engine struct {
	hSocket    *divert.Handle
	hNetwork   *divert.Handle
	channel    chan Packet
	Process    map[uint32]bool
	Tuple      map[Tuple]bool
	TupleMutex sync.Mutex
	writer     io.Writer
	Queue      []Packet
}

func NewEngine() *Engine {
	h1, err := divert.Open(Filter1, divert.LayerSocket, 0, divert.FlagRecvOnly|divert.FlagSniff)
	if err != nil {
		panic(err)
	}

	h2, err := divert.Open(Filter2, divert.LayerNetwork, 1, 0)
	if err != nil {
		panic(err)
	}

	engine := &Engine{
		hSocket:  h1,
		hNetwork: h2,
		channel:  make(chan Packet),
		Process:  make(map[uint32]bool),
		Tuple:    make(map[Tuple]bool),
	}
	engine.writer = NewStack(engine.NetStack_Output)
	return engine
}

func (e *Engine) Start() {
	go e.SocketLayer()
	go e.NetworkLayer()
	go e.PacketHandler()
}

func (e *Engine) SocketLayer() {
	buffer := make([]byte, mtu)
	address := &divert.Address{}
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
			e.TupleMutex.Lock()
			if val {
				e.Tuple[tuple] = true
			} else {
				e.Tuple[tuple] = false
			}
			e.TupleMutex.Unlock()
		} else {
			name, _ := shadow.QueryName(PID)
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
	address := &divert.Address{}
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
			e.TupleMutex.Lock()
			if val, ok := e.Tuple[tuple]; ok {
				if val {
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
			e.TupleMutex.Unlock()
		}
	}
}

func (e *Engine) PacketHandler() {
	timeout := 5 * time.Millisecond
	ticker := time.NewTicker(timeout)
	for {
		select {
		case packet := <-e.channel:
			e.Queue = append(e.Queue, packet)
		case <-ticker.C:
			now := time.Now()
			for _, item := range e.Queue {
				if now.After(item.timestamp.Add(timeout)) {
					if val, ok := e.Tuple[item.tuple]; ok {
						if val {
							e.writer.Write(item.buffer)
							continue
						} else {
						}
					}
					_, err := e.hNetwork.Send(item.buffer, item.address)
					if err != nil {
						log.Printf("[PacketHandler] Failed to send packet: %v\n", err)
					}
				} else {
					break
				}
			}
		}
	}
}

func (e *Engine) NetStack_Output(buffer []byte) (int, error) {
	log.Println("NetStack_Output")
	address := divert.Address{}
	const f = uint8(0x01<<7) | uint8(0x01<<6) | uint8(0x01<<5)
	address.Flags |= f
	address.Network().InterfaceIndex = 7
	size, err := e.hNetwork.Send(buffer, &address)
	return int(size), err
}
