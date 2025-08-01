package main

import (
	"github.com/metacubex/gvisor/pkg/tcpip"
	"github.com/metacubex/gvisor/pkg/tcpip/adapters/gonet"
	"github.com/metacubex/gvisor/pkg/tcpip/header"
	"github.com/metacubex/gvisor/pkg/tcpip/network/ipv4"
	"github.com/metacubex/gvisor/pkg/tcpip/network/ipv6"
	"github.com/metacubex/gvisor/pkg/tcpip/stack"
	"github.com/metacubex/gvisor/pkg/tcpip/transport/icmp"
	"github.com/metacubex/gvisor/pkg/tcpip/transport/tcp"
	"github.com/metacubex/gvisor/pkg/tcpip/transport/udp"
)

const DefaultNIC tcpip.NICID = 1

type GVisor struct {
	stack    *stack.Stack
	endpoint stack.LinkEndpoint
}

func (t *GVisor) Start() error {
	linkEndpoint := t.endpoint
	nicOptions := stack.NICOptions{}
	ipStack, err := NewGVisorStackWithOptions(linkEndpoint, nicOptions)
	if err != nil {
		return err
	}
	ipStack.SetTransportProtocolHandler(tcp.ProtocolNumber, NewTCPForwarderWithLoopback(t.ctx, ipStack, t.handler, t.inet4LoopbackAddress, t.inet6LoopbackAddress, t.tun).HandlePacket)
	ipStack.SetTransportProtocolHandler(udp.ProtocolNumber, NewUDPForwarder(t.ctx, ipStack, t.handler).HandlePacket)

	t.stack = ipStack
	t.endpoint = linkEndpoint
	return nil
}

func NewGVisorStackWithOptions(ep stack.LinkEndpoint, opts stack.NICOptions) (*stack.Stack, error) {
	ipStack := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
			icmp.NewProtocol4,
			icmp.NewProtocol6,
		},
	})
	err := ipStack.CreateNICWithOptions(DefaultNIC, ep, opts)
	if err != nil {
		return nil, gonet.TranslateNetstackError(err)
	}
	ipStack.SetRouteTable([]tcpip.Route{
		{Destination: header.IPv4EmptySubnet, NIC: DefaultNIC},
		{Destination: header.IPv6EmptySubnet, NIC: DefaultNIC},
	})
	ipStack.SetSpoofing(DefaultNIC, true)
	ipStack.SetPromiscuousMode(DefaultNIC, true)
	bufSize := 20 * 1024
	ipStack.SetTransportProtocolOption(tcp.ProtocolNumber, &tcpip.TCPReceiveBufferSizeRangeOption{
		Min:     1,
		Default: bufSize,
		Max:     bufSize,
	})
	ipStack.SetTransportProtocolOption(tcp.ProtocolNumber, &tcpip.TCPSendBufferSizeRangeOption{
		Min:     1,
		Default: bufSize,
		Max:     bufSize,
	})
	sOpt := tcpip.TCPSACKEnabled(true)
	ipStack.SetTransportProtocolOption(tcp.ProtocolNumber, &sOpt)
	mOpt := tcpip.TCPModerateReceiveBufferOption(true)
	ipStack.SetTransportProtocolOption(tcp.ProtocolNumber, &mOpt)
	return ipStack, nil
}

func main() {

}
