//go:build windows

package ui

import (
	"syscall"
	"unicode/utf16"
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_MINIMIZEBOX      = 0x00020000
	WS_CLIPCHILDREN     = 0x02000000
	WS_VISIBLE          = 0x10000000
	WS_EX_TOPMOST       = 0x00000008
	WS_EX_TOOLWINDOW    = 0x00000080
	WS_EX_APPWINDOW     = 0x00040000
	CS_HREDRAW          = 0x0002
	CS_VREDRAW          = 0x0001
	CS_DBLCLKS          = 0x0008
	SW_HIDE             = 0
	SW_SHOWNORMAL       = 1
	SW_SHOW             = 5
	SW_RESTORE          = 9

	WM_CREATE      = 0x0001
	WM_DESTROY     = 0x0002
	WM_SIZE        = 0x0005
	WM_PAINT       = 0x000F
	WM_CLOSE       = 0x0010
	WM_ERASEBKGND  = 0x0014
	WM_SETCURSOR   = 0x0020
	WM_DPICHANGED  = 0x02E0
	WM_COMMAND     = 0x0111
	WM_TIMER       = 0x0113
	WM_MOUSEMOVE      = 0x0200
	WM_LBUTTONDOWN    = 0x0201
	WM_LBUTTONUP      = 0x0202
	WM_LBUTTONDBLCLK  = 0x0203
	WM_RBUTTONUP      = 0x0205
	WM_MBUTTONUP      = 0x0208
	WM_MOUSEWHEEL     = 0x020A
	WM_CAPTURECHANGED = 0x0215
	WM_KEYDOWN        = 0x0100
	WM_HOTKEY         = 0x0312
	VK_PRIOR          = 0x21
	VK_NEXT           = 0x22
	VK_END            = 0x23
	VK_HOME           = 0x24
	VK_UP             = 0x26
	VK_DOWN           = 0x28
	WM_USER        = 0x0400
	WM_TRAYICON    = WM_USER + 42

	NIM_ADD, NIM_MODIFY, NIM_DELETE, NIM_SETVERSION = 0, 1, 2, 4
	NIF_MESSAGE, NIF_ICON, NIF_TIP, NIF_INFO        = 1, 2, 4, 16
	NIIF_INFO, NIIF_NOSOUND                         = 1, 16
	TPM_RIGHTBUTTON, TPM_RETURNCMD                  = 0x0002, 0x0100
	MF_STRING, MF_SEPARATOR, MF_POPUP               = 0x0000, 0x0800, 0x0010
	MF_CHECKED, MF_GRAYED                           = 0x0008, 0x0001
	DWMWA_USE_IMMERSIVE_DARK_MODE                   = 20
	DWMWA_WINDOW_CORNER_PREFERENCE                  = 33
	DWMWCP_ROUND                                    = 2
	IDC_ARROW, IDC_HAND                             = 32512, 32649
	CC_RGBINIT, CC_FULLOPEN, CC_ANYCOLOR            = 0x0001, 0x0002, 0x0100
)

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type PAINTSTRUCT struct {
	HDC         uintptr
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type NOTIFYICONDATAW struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

type ICONINFO struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

type CHOOSECOLORW struct {
	LStructSize    uint32
	HwndOwner      uintptr
	HInstance      uintptr
	RgbResult      uint32
	LpCustColors   *uint32
	Flags          uint32
	LCustData      uintptr
	LpfnHook       uintptr
	LpTemplateName *uint16
}

// UTF16Ptr converte uma string Go para ponteiro UTF-16 terminado em nulo.
func UTF16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		empty := uint16(0)
		return &empty
	}
	return p
}

// CopyUTF16 copia s para o slice UTF-16 de destino garantindo terminador nulo.
func CopyUTF16(dst []uint16, s string) {
	if len(dst) == 0 {
		return
	}
	encoded := utf16.Encode([]rune(s))
	maxLen := len(dst) - 1
	if len(encoded) > maxLen {
		encoded = encoded[:maxLen]
	}
	copy(dst, encoded)
	for i := len(encoded); i < len(dst); i++ {
		dst[i] = 0
	}
}
