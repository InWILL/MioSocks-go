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
}

func Open(filter string, layer Layer, priority int16, flags uint64) (h *Handle, err error) {
	filterPtr, err := windows.BytePtrFromString(filter)
	if err != nil {
		return nil, err
	}

	hPtr, _, err := windivertOpen.Call(
		uintptr(unsafe.Pointer(filterPtr)),
		uintptr(layer),
		uintptr(priority),
		uintptr(flags))
	if err != nil {
		return nil, err
	}

	return &Handle{
		Handle: windows.Handle(hPtr),
	}, nil
}
