package engine

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

func GetProcName(pid uint32) (string, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)

	buffer := make([]uint16, windows.MAX_PATH)
	size := uint32(windows.MAX_PATH)

	err = windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size)
	if err != nil {
		return "", err
	}

	fullname := windows.UTF16ToString(buffer[:size])
	_, name := filepath.Split(fullname)
	return name, nil
}
