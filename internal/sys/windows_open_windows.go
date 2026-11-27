//go:build windows

package sys

import (
	"sync"
	"syscall"
	"unsafe"
)

const (
	desktopReadObjects    = 0x0001
	desktopEnumerate      = 0x0040
	dwmwaCloaked          = 14
	dwmCloakedApp         = 0x00000001
	dwmCloakedInherited   = 0x00000004
	procCommandLineInfo   = 60
	statusInfoLenMismatch = 0xC0000004
	statusBufferTooSmall  = 0xC0000023
	statusBufferOverflow  = 0x80000005
)

type unicodeString struct {
	Length        uint16
	MaximumLength uint16
	_             uint32
	Buffer        uintptr
}

var (
	enumMu     sync.Mutex
	activePIDs map[uint32]bool

	cbTopWindow   = syscall.NewCallback(enumTopWindowProc)
	cbChildWindow = syscall.NewCallback(enumChildWindowProc)
)

func readWindowTitle(hwnd uintptr) string {
	n, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if n == 0 {
		return ""
	}
	if n > 256 {
		n = 256
	}
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func isIgnoredShellTitle(title string, cloaked uint32) bool {
	if title == "" || title == "Program Manager" || title == "Windows Input Experience" {
		return true
	}
	if cloaked != 0 {
		switch title {
		case "Settings", "Configurações", "Microsoft Text Input Application", "Cortana", "Search":
			return true
		}
	}
	return false
}

func enumChildWindowProc(hwnd, _ uintptr) uintptr {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid > 4 && activePIDs != nil {
		activePIDs[pid] = true
	}
	return 1
}

func enumTopWindowProc(hwnd, _ uintptr) uintptr {
	vis, _, _ := procIsWindowVisible.Call(hwnd)
	if vis == 0 {
		return 1
	}
	var cloaked uint32
	procDwmGetWindowAttribute.Call(hwnd, dwmwaCloaked, uintptr(unsafe.Pointer(&cloaked)), 4)
	// Preserva janelas em outra Area de Trabalho Virtual (DWM_CLOAKED_SHELL=0x2); descarta apps suspensos (0x1/0x4).
	if cloaked&(dwmCloakedApp|dwmCloakedInherited) != 0 {
		return 1
	}
	title := readWindowTitle(hwnd)
	if isIgnoredShellTitle(title, cloaked) {
		return 1
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid > 4 && activePIDs != nil {
		activePIDs[pid] = true
		procEnumChildWindows.Call(hwnd, cbChildWindow, 0)
	}
	return 1
}

// EnumOpenWindowPIDs detecta PIDs de aplicativos com janelas visiveis no desktop interativo.
func EnumOpenWindowPIDs() (map[uint32]bool, bool) {
	enumMu.Lock()
	defer enumMu.Unlock()

	activePIDs = make(map[uint32]bool)
	defer func() { activePIDs = nil }()

	deskPtr, err := syscall.UTF16PtrFromString("Default")
	if err == nil {
		hDesk, _, _ := procOpenDesktopW.Call(uintptr(unsafe.Pointer(deskPtr)), 0, 0, desktopReadObjects|desktopEnumerate)
		if hDesk != 0 {
			procEnumDesktopWindows.Call(hDesk, cbTopWindow, 0)
			procCloseDesktop.Call(hDesk)
		}
	}
	procEnumWindows.Call(cbTopWindow, 0)

	out := make(map[uint32]bool, len(activePIDs))
	for pid := range activePIDs {
		out[pid] = true
	}
	return out, len(out) > 0
}

// ReadProcessCmdLine le a linha de comando de um processo via NtQueryInformationProcess.
func ReadProcessCmdLine(hProc uintptr) string {
	if hProc == 0 {
		return ""
	}
	buf := make([]byte, 4096)
	var retLen uint32
	st, _, _ := procNtQueryInformationProcess.Call(
		hProc,
		procCommandLineInfo,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&retLen)),
	)
	if (uint32(st) == statusInfoLenMismatch || uint32(st) == statusBufferTooSmall || uint32(st) == statusBufferOverflow) && retLen > 0 && retLen <= 65536 {
		buf = make([]byte, retLen)
		st, _, _ = procNtQueryInformationProcess.Call(
			hProc,
			procCommandLineInfo,
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(len(buf)),
			uintptr(unsafe.Pointer(&retLen)),
		)
	}
	if uint32(st) != 0 || len(buf) < int(unsafe.Sizeof(unicodeString{})) {
		return ""
	}
	us := (*unicodeString)(unsafe.Pointer(&buf[0]))
	if us.Length == 0 || us.Buffer == 0 {
		return ""
	}
	baseAddr := uintptr(unsafe.Pointer(&buf[0]))
	endAddr := baseAddr + uintptr(len(buf))
	if us.Buffer < baseAddr || us.Buffer+uintptr(us.Length) > endAddr {
		return ""
	}
	off := int(us.Buffer - baseAddr)
	if off < 0 || off+int(us.Length) > len(buf) {
		return ""
	}
	u16 := unsafe.Slice((*uint16)(unsafe.Pointer(&buf[off])), int(us.Length/2))
	return syscall.UTF16ToString(u16)
}
