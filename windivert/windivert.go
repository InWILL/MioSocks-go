package windivert

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

type Handle struct {
	windows.Handle
}

var (
	windivertDLL  = (*windows.DLL)(nil)
	windivertOpen = (*windows.Proc)(nil)
	windivertRecv = (*windows.Proc)(nil)
	windivertSend = (*windows.Proc)(nil)
)

func init() {
	dll, err := windows.LoadDLL("WinDivert.dll")
	if err != nil {
		panic(err)
	}
	windivertDLL = dll

	proc, err := windivertDLL.FindProc("WinDivertOpen")
	if err != nil {
		panic(err)
	}
	windivertOpen = proc

	proc, err = windivertDLL.FindProc("WinDivertRecv")
	if err != nil {
		panic(err)
	}
	windivertRecv = proc

	proc, err = windivertDLL.FindProc("WinDivertSend")
	if err != nil {
		panic(err)
	}
	windivertSend = proc
}

func Open(filter string, layer Layer, priority int16, flags uint64) (h *Handle, err error) {
	filterPtr, err := windows.BytePtrFromString(filter)
	if err != nil {
		return nil, err
	}

	handle, _, err := windivertOpen.Call(
		uintptr(unsafe.Pointer(filterPtr)),
		uintptr(layer),
		uintptr(priority),
		uintptr(flags))
	if windows.Handle(handle) == windows.InvalidHandle {
		return nil, err
	}

	return &Handle{
		Handle: windows.Handle(handle),
	}, nil
}

func (h *Handle) Recv(buffer []byte, address *Address) (int, error) {
	var length int
	windivertRecv.Call(
		uintptr(h.Handle),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
		uintptr(unsafe.Pointer(&length)),
		uintptr(unsafe.Pointer(address)),
	)
	return length, nil
}

func (h *Handle) Send(buffer []byte, address *Address) (int, error) {
	var length int
	windivertSend.Call(
		uintptr(h.Handle),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
		uintptr(unsafe.Pointer(&length)),
		uintptr(unsafe.Pointer(address)),
	)
	return length, nil
}
