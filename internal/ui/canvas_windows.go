//go:build windows

package ui

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

var (
	procCreateCompatibleDC = sys.Gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC           = sys.Gdi32.NewProc("DeleteDC")
	procCreateDIBSection   = sys.Gdi32.NewProc("CreateDIBSection")
	procSelectObject       = sys.Gdi32.NewProc("SelectObject")
	procDeleteObject       = sys.Gdi32.NewProc("DeleteObject")
	procBitBlt             = sys.Gdi32.NewProc("BitBlt")
	procGdiFlush           = sys.Gdi32.NewProc("GdiFlush")
	procSetBkMode          = sys.Gdi32.NewProc("SetBkMode")
	procSetTextColor       = sys.Gdi32.NewProc("SetTextColor")
	procCreateFontW        = sys.Gdi32.NewProc("CreateFontW")
	procCreateBitmap       = sys.Gdi32.NewProc("CreateBitmap")
	procGetDC              = sys.User32.NewProc("GetDC")
	procReleaseDC          = sys.User32.NewProc("ReleaseDC")
	procDrawTextW          = sys.User32.NewProc("DrawTextW")
	procCreateIconIndirect = sys.User32.NewProc("CreateIconIndirect")
	procDestroyIcon        = sys.User32.NewProc("DestroyIcon")
)

const (
	dibRGBColors        = 0
	biRGB               = 0
	srcCopy             = 0x00CC0020
	transparentBkMode   = 1
	cleartypeQuality    = 5
	antialiasedQuality  = 4
	dtLeft              = 0x00000000
	dtCenter            = 0x00000001
	dtRight             = 0x00000002
	dtVCenter           = 0x00000004
	dtSingleLine        = 0x00000020
	dtEndEllipsis       = 0x00008000
	dtNoPrefix          = 0x00000800
)

// Canvas fornece buffer DIB 32bpp top-down para desenho anti-aliased e texto ClearType.
type Canvas struct {
	Width  int
	Height int
	HDC    uintptr
	HBitmap uintptr
	oldBmp uintptr
	pixels []byte
}

// NewCanvas cria um buffer offscreen pronto para pixel ops e DrawTextW.
func NewCanvas(w, h int) *Canvas {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	screenDC, _, _ := procGetDC.Call(0)
	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if screenDC != 0 {
		procReleaseDC.Call(0, screenDC)
	}

	bmi := BITMAPINFO{
		BmiHeader: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(w),
			BiHeight:      -int32(h),
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: biRGB,
		},
	}
	var bits unsafe.Pointer
	hBmp, _, _ := procCreateDIBSection.Call(memDC, uintptr(unsafe.Pointer(&bmi)), dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	oldBmp, _, _ := procSelectObject.Call(memDC, hBmp)
	var pix []byte
	if bits != nil {
		pix = unsafe.Slice((*byte)(bits), w*h*4)
	}
	return &Canvas{Width: w, Height: h, HDC: memDC, HBitmap: hBmp, oldBmp: oldBmp, pixels: pix}
}

// Close libera todos os recursos GDI do Canvas.
func (c *Canvas) Close() {
	if c == nil {
		return
	}
	if c.HDC != 0 && c.oldBmp != 0 {
		procSelectObject.Call(c.HDC, c.oldBmp)
		c.oldBmp = 0
	}
	if c.HBitmap != 0 {
		procDeleteObject.Call(c.HBitmap)
		c.HBitmap = 0
	}
	if c.HDC != 0 {
		procDeleteDC.Call(c.HDC)
		c.HDC = 0
	}
	c.pixels = nil
}

// ClearOpaque preenche todo o buffer com uma cor ARGB opaca.
func (c *Canvas) ClearOpaque(argb uint32) {
	b := byte(argb)
	g := byte(argb >> 8)
	r := byte(argb >> 16)
	for i := 0; i+3 < len(c.pixels); i += 4 {
		c.pixels[i] = b
		c.pixels[i+1] = g
		c.pixels[i+2] = r
		c.pixels[i+3] = 255
	}
}

// DrawText desenha texto ClearType no Canvas preservando alinhamento horizontal/vertical.
func (c *Canvas) DrawText(text, fontName string, heightPx int, bold bool, argb uint32, rect RECT, flags uint32) {
	if c.HDC == 0 || text == "" {
		return
	}
	weight := uintptr(400)
	if bold {
		weight = 700
	}
	hFont, _, _ := procCreateFontW.Call(
		uintptr(^uintptr(uint(heightPx)-1)), // -heightPx em complemento de dois
		0, 0, 0, weight, 0, 0, 0, 1, 0, 0, cleartypeQuality, 0,
		uintptr(unsafe.Pointer(UTF16Ptr(fontName))),
	)
	if hFont == 0 {
		return
	}
	defer procDeleteObject.Call(hFont)

	oldFont, _, _ := procSelectObject.Call(c.HDC, hFont)
	defer procSelectObject.Call(c.HDC, oldFont)

	procSetBkMode.Call(c.HDC, transparentBkMode)
	procSetTextColor.Call(c.HDC, uintptr(ARGBToCOLORREF(argb)))
	u16 := UTF16Ptr(text)
	procDrawTextW.Call(c.HDC, uintptr(unsafe.Pointer(u16)), ^uintptr(0), uintptr(unsafe.Pointer(&rect)), uintptr(flags|dtNoPrefix))
	procGdiFlush.Call()
}

// BlitTo copia o Canvas inteiro para o HDC da janela alvo.
func (c *Canvas) BlitTo(targetHDC uintptr) {
	procGdiFlush.Call()
	procBitBlt.Call(targetHDC, 0, 0, uintptr(c.Width), uintptr(c.Height), c.HDC, 0, 0, srcCopy)
}

// ToNRGBA converte o buffer BGRA do DIB em *image.NRGBA opaco para exportacao PNG.
func (c *Canvas) ToNRGBA(forceOpaque bool) *image.NRGBA {
	procGdiFlush.Call()
	img := image.NewNRGBA(image.Rect(0, 0, c.Width, c.Height))
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			i := (y*c.Width + x) * 4
			a := c.pixels[i+3]
			if forceOpaque {
				a = 255
			}
			img.Pix[i] = c.pixels[i+2]
			img.Pix[i+1] = c.pixels[i+1]
			img.Pix[i+2] = c.pixels[i]
			img.Pix[i+3] = a
		}
	}
	return img
}

// SavePNG grava o conteudo atual do Canvas como imagem PNG.
func (c *Canvas) SavePNG(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, c.ToNRGBA(true))
}
