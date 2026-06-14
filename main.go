package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"GoWork/config"
	"GoWork/ui"
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	setProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	freeConsole        = kernel32.NewProc("FreeConsole")
)

func init() {
	setProcessDPIAware.Call()
	freeConsole.Call()
}

func initCommonControls() {
	commctl32 := syscall.NewLazyDLL("comctl32.dll")
	procInitCommonControlsEx := commctl32.NewProc("InitCommonControlsEx")

	type tagINITCOMMONCONTROLSEX struct {
		dwSize uint32
		dwICC  uint32
	}

	icc := tagINITCOMMONCONTROLSEX{
		dwSize: uint32(unsafe.Sizeof(tagINITCOMMONCONTROLSEX{})),
		dwICC:  0x00004000 | 0x000000FF,
	}

	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
}

func main() {
	// Panic recovery: write to file instead of stdout
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			msg := fmt.Sprintf("=== 程序异常 ===\n%v\n%s\n================\n", r, buf[:n])
			// Write to crash.log instead of stdout to avoid console flash
			os.WriteFile("crash.log", []byte(msg), 0644)
		}
	}()

	initCommonControls()

	cfg := config.Load()
	app := ui.NewApp(cfg)
	app.Run()
	app.Cleanup()
}
