//go:build windows

package ui

import (
	"fmt"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

const (
	wmAppShow      = WM_USER + 101
	wmAppCleanDone = WM_USER + 102

	swpNoMove     = 0x0002
	swpNoSize     = 0x0001
	swpNoActivate = 0x0010
	hwndTopMost   = ^uintptr(0)
	hwndNoTopMost = ^uintptr(1)

	mbYesNo           = 0x00000004
	mbIconWarning     = 0x00000030
	mbIconInformation = 0x00000040
	idYes             = 6
)

var (
	procRegisterClassExW       = sys.User32.NewProc("RegisterClassExW")
	procCreateWindowExW        = sys.User32.NewProc("CreateWindowExW")
	procDefWindowProcW         = sys.User32.NewProc("DefWindowProcW")
	procShowWindow             = sys.User32.NewProc("ShowWindow")
	procUpdateWindow           = sys.User32.NewProc("UpdateWindow")
	procInvalidateRect         = sys.User32.NewProc("InvalidateRect")
	procBeginPaint             = sys.User32.NewProc("BeginPaint")
	procEndPaint               = sys.User32.NewProc("EndPaint")
	procPostQuitMessage        = sys.User32.NewProc("PostQuitMessage")
	procGetMessageW            = sys.User32.NewProc("GetMessageW")
	procTranslateMessage       = sys.User32.NewProc("TranslateMessage")
	procDispatchMessageW       = sys.User32.NewProc("DispatchMessageW")
	procSetTimer               = sys.User32.NewProc("SetTimer")
	procKillTimer              = sys.User32.NewProc("KillTimer")
	procRegisterHotKey         = sys.User32.NewProc("RegisterHotKey")
	procUnregisterHotKey       = sys.User32.NewProc("UnregisterHotKey")
	procSetWindowPos           = sys.User32.NewProc("SetWindowPos")
	procGetDpiForWindow        = sys.User32.NewProc("GetDpiForWindow")
	procLoadCursorW            = sys.User32.NewProc("LoadCursorW")
	procSetCursor              = sys.User32.NewProc("SetCursor")
	procRegisterWindowMessageW = sys.User32.NewProc("RegisterWindowMessageW")
	procMessageBoxW            = sys.User32.NewProc("MessageBoxW")
	procFindWindowW            = sys.User32.NewProc("FindWindowW")
	procIsWindowVisible        = sys.User32.NewProc("IsWindowVisible")
	procAdjustWindowRectExForDpi = sys.User32.NewProc("AdjustWindowRectExForDpi")
	procDwmSetWindowAttribute  = sys.Dwmapi.NewProc("DwmSetWindowAttribute")
	procChooseColorW           = sys.Comdlg32.NewProc("ChooseColorW")
	procGetModuleHandleW       = sys.Kernel32.NewProc("GetModuleHandleW")
	procCreateMutexW           = sys.Kernel32.NewProc("CreateMutexW")
)

// HitRegion mapeia uma area interativa na tela renderizada via GDI.
type HitRegion struct {
	ID   string
	Rect RECT
	Arg  uint32
}

// FindHit retorna o HitRegion sob (x, y), priorizando o ultimo desenhado.
func FindHit(hits []HitRegion, x, y int32) (HitRegion, bool) {
	for i := len(hits) - 1; i >= 0; i-- {
		r := hits[i].Rect
		if x >= r.Left && x < r.Right && y >= r.Top && y < r.Bottom {
			return hits[i], true
		}
	}
	return HitRegion{}, false
}

// ApplyWindowDarkChrome ativa barra escura e cantos arredondados nativos no Windows 11.
func ApplyWindowDarkChrome(hwnd uintptr) {
	dark := int32(1)
	procDwmSetWindowAttribute.Call(hwnd, DWMWA_USE_IMMERSIVE_DARK_MODE, uintptr(unsafe.Pointer(&dark)), 4)
	round := int32(DWMWCP_ROUND)
	procDwmSetWindowAttribute.Call(hwnd, DWMWA_WINDOW_CORNER_PREFERENCE, uintptr(unsafe.Pointer(&round)), 4)
}

// ApplyTopMost alterna o estado Sempre Visivel no Topo.
func ApplyTopMost(hwnd uintptr, onTop bool) {
	insertAfter := hwndNoTopMost
	if onTop {
		insertAfter = hwndTopMost
	}
	procSetWindowPos.Call(hwnd, insertAfter, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
}

// ConfirmBox exibe caixa de dialogo nativa Sim/Nao com seguranca.
func ConfirmBox(hwnd uintptr, title, text string) bool {
	ret, _, _ := procMessageBoxW.Call(
		hwnd,
		uintptr(unsafe.Pointer(UTF16Ptr(text))),
		uintptr(unsafe.Pointer(UTF16Ptr(title))),
		mbYesNo|mbIconWarning,
	)
	return ret == idYes
}

// ChooseRGBColor abre o seletor nativo de cores do Windows retornando 0x00RRGGBB.
func ChooseRGBColor(hwnd uintptr, initialRGB uint32) (uint32, bool) {
	var cust [16]uint32
	cust[0] = ARGBToCOLORREF(RGBToARGB(0x10B981, 255))
	cust[1] = ARGBToCOLORREF(RGBToARGB(0xF59E0B, 255))
	cust[2] = ARGBToCOLORREF(RGBToARGB(0xEF4444, 255))
	cust[3] = ARGBToCOLORREF(RGBToARGB(0xFFFFFF, 255))

	cc := CHOOSECOLORW{
		LStructSize:  uint32(unsafe.Sizeof(CHOOSECOLORW{})),
		HwndOwner:    hwnd,
		RgbResult:    ARGBToCOLORREF(RGBToARGB(initialRGB, 255)),
		LpCustColors: &cust[0],
		Flags:        CC_RGBINIT | CC_FULLOPEN | CC_ANYCOLOR,
	}
	ok, _, _ := procChooseColorW.Call(uintptr(unsafe.Pointer(&cc)))
	if ok == 0 {
		return initialRGB, false
	}
	argb := COLORREFToARGB(cc.RgbResult, 255)
	return argb & 0x00FFFFFF, true
}

func drawButton(c *Canvas, id string, x, y, w, h int, label string, bg, border, fg uint32, bold bool, hoverID string) HitRegion {
	fill := bg
	if hoverID == id {
		fill = blendLighter(bg)
	}
	c.FillRoundRect(x, y, w, h, RadiusButton, fill)
	if border != 0 {
		c.StrokeRoundRect(x, y, w, h, RadiusButton, 1, border)
	}
	c.DrawText(label, FontUI, 13, bold, fg, RECT{int32(x), int32(y), int32(x + w), int32(y + h)}, dtCenter|dtVCenter|dtSingleLine)
	return HitRegion{ID: id, Rect: RECT{int32(x), int32(y), int32(x + w), int32(y + h)}}
}

func blendLighter(argb uint32) uint32 {
	a := argb >> 24
	r := min(255, int((argb>>16)&0xFF)+18)
	g := min(255, int((argb>>8)&0xFF)+18)
	b := min(255, int(argb&0xFF)+18)
	return (a << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
}

func drawCheckbox(c *Canvas, id string, x, y int, checked bool, label string, hoverID string) HitRegion {
	boxSize := 16
	fill := ColorBgElevated
	border := ColorBorder
	if checked {
		fill = ColorGreen
		border = ColorGreen
	} else if hoverID == id {
		border = ColorTextMuted
	}
	c.FillRoundRect(x, y+2, boxSize, boxSize, 4, fill)
	c.StrokeRoundRect(x, y+2, boxSize, boxSize, 4, 1, border)
	if checked {
		c.DrawCheckmark(x, y+2, boxSize, ColorBgBase)
	}
	c.DrawText(label, FontUI, 13, false, ColorTextPrimary, RECT{int32(x + 24), int32(y), int32(x + 380), int32(y + 22)}, dtLeft|dtVCenter|dtSingleLine)
	return HitRegion{ID: id, Rect: RECT{int32(x), int32(y), int32(x + 360), int32(y + 22)}}
}

func drawStepper(c *Canvas, idPrefix string, x, y int, value int, suffix string, hoverID string) []HitRegion {
	var hits []HitRegion
	minus := drawButton(c, idPrefix+"_dec", x, y, 26, 24, "-", ColorBgElevated, ColorBorder, ColorTextPrimary, true, hoverID)
	c.FillRoundRect(x+30, y, 56, 24, RadiusControl, ColorBgBase)
	c.StrokeRoundRect(x+30, y, 56, 24, RadiusControl, 1, ColorBorder)
	c.DrawText(fmt.Sprintf("%d%s", value, suffix), FontUI, 12, true, ColorTextPrimary, RECT{int32(x + 30), int32(y), int32(x + 86), int32(y + 24)}, dtCenter|dtVCenter|dtSingleLine)
	plus := drawButton(c, idPrefix+"_inc", x+90, y, 26, 24, "+", ColorBgElevated, ColorBorder, ColorTextPrimary, true, hoverID)
	hits = append(hits, minus, plus)
	return hits
}
