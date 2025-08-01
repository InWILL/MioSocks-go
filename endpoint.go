package main

import (
	"io"

	"github.com/metacubex/gvisor/pkg/buffer"
	"github.com/metacubex/gvisor/pkg/tcpip"
	"github.com/metacubex/gvisor/pkg/tcpip/header"
	"github.com/metacubex/gvisor/pkg/tcpip/stack"
)

type Endpoint struct {
	device     io.ReadWriter
	dispatcher stack.NetworkDispatcher
}

func NewEndpoint(d io.ReadWriter) stack.LinkEndpoint {
	return &Endpoint{device: d}
}

// stack.LinkEndpoint default interface
func (e *Endpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (e *Endpoint) AddHeader(*stack.PacketBuffer) {

}

func (e *Endpoint) Attach(dispatcher stack.NetworkDispatcher) {
	if dispatcher == nil && e.dispatcher != nil {
		e.dispatcher = nil
		return
	}
	if dispatcher != nil && e.dispatcher == nil {
		e.dispatcher = dispatcher
		go e.dispatchLoop()
	}
}

func (e *Endpoint) Capabilities() stack.LinkEndpointCapabilities {
	return stack.CapabilityRXChecksumOffload
}

func (e *Endpoint) Close() {

}

func (e *Endpoint) IsAttached() bool {
	return e.dispatcher != nil
}

func (e *Endpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

func (e *Endpoint) MTU() uint32 {
	return 1500
}

func (e *Endpoint) MaxHeaderLength() uint16 {
	return 0
}

func (e *Endpoint) ParseHeader(*stack.PacketBuffer) bool {
	return true
}

func (e *Endpoint) SetLinkAddress(addr tcpip.LinkAddress) {

}

func (e *Endpoint) SetMTU(mtu uint32) {}

func (e *Endpoint) SetOnCloseAction(func()) {

}

func (e *Endpoint) Wait() {

}

func (e *Endpoint) WritePackets(stack.PacketBufferList) (int, tcpip.Error)

// stack.LinkEndpoint personal interface
func (e *Endpoint) dispatchLoop() {
	for {
		buf := make([]byte, 1500)
		n, err := e.device.Read(buf)
		if err != nil {
			break
		}
		packetBuffer := buffer.MakeWithData(buf[:n])
		ihl, ok := packetBuffer.PullUp(0, 1)
		if !ok {
			packetBuffer.Release()
			continue
		}
		var networkProtocol tcpip.NetworkProtocolNumber
		switch header.IPVersion(ihl.AsSlice()) {
		case header.IPv4Version:
			networkProtocol = header.IPv4ProtocolNumber
		case header.IPv6Version:
			networkProtocol = header.IPv6ProtocolNumber
		default:
			e.device.Write(packetBuffer.Flatten())
			packetBuffer.Release()
			continue
		}
		pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
			Payload:           packetBuffer,
			IsForwardedPacket: true,
		})
		dispatcher := e.dispatcher
		if dispatcher == nil {
			pkt.DecRef()
			break
		}
		dispatcher.DeliverNetworkPacket(networkProtocol, pkt)
		pkt.DecRef()
	}
}
