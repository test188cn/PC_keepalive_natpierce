package ui

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

// loadHIconFromFile loads an ICO file using Windows LoadImage API.
func loadHIconFromFile(path string) (win.HICON, error) {
	user32 := syscall.NewLazyDLL("user32.dll")
	procLoadImage := user32.NewProc("LoadImageW")

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	const IMAGE_ICON = 1
	const LR_LOADFROMFILE = 0x0010
	const LR_DEFAULTSIZE = 0x0040

	hIcon, _, _ := procLoadImage.Call(
		0,
		uintptr(unsafe.Pointer(pathPtr)),
		IMAGE_ICON,
		0, 0,
		LR_LOADFROMFILE|LR_DEFAULTSIZE,
	)

	if hIcon == 0 {
		return 0, fmt.Errorf("LoadImage failed for %s", path)
	}

	return win.HICON(hIcon), nil
}
