//go:build windows

package ui

import (
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

const mainClassName = "RamKillerMainClass"

func loadAppIcon(hInst uintptr, size int) uintptr {
	ic, _, _ := procLoadImageW.Call(hInst, 1, 1 /* IMAGE_ICON */, uintptr(size), uintptr(size), 0)
	return ic
}

func createFixedDarkWindow(className, title string, clientW, clientH int, wndProc uintptr, visible bool) uintptr {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	clsPtr := UTF16Ptr(className)
	cur, _, _ := procLoadCursorW.Call(0, IDC_ARROW)
	iconLg := loadAppIcon(hInst, 32)
	iconSm := loadAppIcon(hInst, 16)

	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         CS_HREDRAW | CS_VREDRAW | CS_DBLCLKS,
		LpfnWndProc:   wndProc,
		HInstance:     hInst,
		HIcon:         iconLg,
		HIconSm:       iconSm,
		HCursor:       cur,
		LpszClassName: clsPtr,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	style := uint32(WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX)
	if visible {
		style |= WS_VISIBLE
	}
	rc := RECT{0, 0, int32(clientW), int32(clientH)}
	procAdjustWindowRectExForDpi.Call(uintptr(unsafe.Pointer(&rc)), uintptr(style), 0, 0, 96)

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(clsPtr)),
		uintptr(unsafe.Pointer(UTF16Ptr(title))),
		uintptr(style),
		120, 100,
		uintptr(rc.Right-rc.Left),
		uintptr(rc.Bottom-rc.Top),
		0, 0, hInst, 0,
	)
	if hwnd != 0 {
		ApplyWindowDarkChrome(hwnd)
		procChangeWindowMsgFilterEx.Call(hwnd, wmAppShow, 1 /* MSGFLT_ALLOW */, 0)
	}
	return hwnd
}

func (a *AppController) mainWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	if a.taskbarCreatedMsg != 0 && uint32(msg) == a.taskbarCreatedMsg {
		a.forceTrayRefresh()
		return 0
	}
	switch msg {
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			cfg := a.getConfig()
			canvas, hits := RenderMainCanvas(a.stats, cfg, a.lastSummary, a.mainHover, a.engine.IsCleaning())
			a.mainHits = hits
			canvas.BlitTo(hdc)
			canvas.Close()
			procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		}
		return 0
	case WM_MOUSEMOVE:
		x := int32(int16(lParam & 0xFFFF))
		y := int32(int16((lParam >> 16) & 0xFFFF))
		if hit, ok := FindHit(a.mainHits, x, y); ok {
			if a.mainHover != hit.ID {
				a.mainHover = hit.ID
				procInvalidateRect.Call(hwnd, 0, 0)
			}
			cur, _, _ := procLoadCursorW.Call(0, IDC_HAND)
			procSetCursor.Call(cur)
		} else if a.mainHover != "" {
			a.mainHover = ""
			procInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case WM_LBUTTONUP:
		x := int32(int16(lParam & 0xFFFF))
		y := int32(int16((lParam >> 16) & 0xFFFF))
		if hit, ok := FindHit(a.mainHits, x, y); ok {
			a.handleMainClick(hit)
		}
		return 0
	case WM_TIMER:
		a.onTick()
		return 0
	case WM_HOTKEY:
		a.triggerClean(domain.SourceHotkey, 0)
		return 0
	case wmAppShow:
		a.showMainWindow()
		return 0
	case wmAppCleanDone:
		a.onCleanFinished()
		return 0
	case WM_TRAYICON:
		a.handleTrayEvent(uint32(lParam & 0xFFFF))
		return 0
	case WM_CLOSE:
		procShowWindow.Call(hwnd, SW_HIDE)
		sys.TrimSelf()
		return 0
	case WM_DESTROY:
		procKillTimer.Call(hwnd, 1)
		procUnregisterHotKey.Call(hwnd, 1)
		if a.tray != nil {
			a.tray.Destroy()
		}
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return ret
}
