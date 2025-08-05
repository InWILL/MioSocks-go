package engine

import (
	"log"

	"github.com/imgk/divert-go"
	shadow "github.com/imgk/shadow/utils"
)

type Engine struct {
	hSocket  *divert.Handle
	hNetwork *divert.Handle
}

const (
	Filter1 = "outbound and !loopback and !ipv6 and (tcp) and event == CONNECT"
	Filter2 = "outbound and !loopback and !ipv6 and (tcp)"
	mtu     = 1500
)

func NewEngine() *Engine {
	h1, err := divert.Open(Filter1, divert.LayerSocket, 0, divert.FlagRecvOnly|divert.FlagSniff)
	if err != nil {
		panic(err)
	}

	//h2, err := divert.Open(Filter2, divert.LayerNetwork, 1, 0)
	if err != nil {
		panic(err)
	}

	engine := &Engine{
		hSocket:  h1,
		hNetwork: nil,
	}
	return engine
}

func (e *Engine) Start() {
	go e.SocketLayer()
}

func (e *Engine) SocketLayer() {
	packet := make([]byte, mtu)
	address := &divert.Address{}
	for {
		_, err := e.hSocket.Recv(packet, address)
		if err != nil {
			log.Printf("[NetworkLayer] Failed to receive packet: %v\n", err)
		}

		name, _ := shadow.QueryName(address.Socket().ProcessID)
		log.Printf("Program:%s PID:%d %d:%d", name, address.Socket().ProcessID, address.Socket().LocalPort, address.Socket().RemotePort)
	}
}

func (e *Engine) NetworkLayer() {

}

func (e *Engine) PacketQueue() {

}
