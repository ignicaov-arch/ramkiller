//go:build windows

package ui

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

func resolveTrayColors(percent int, cfg domain.Config) (bg uint32, txt uint32, border uint32, transparent bool) {
	sev := domain.SeverityForPercent(percent, cfg)
	accent := cfg.ColorBg
	if sev == domain.SeverityWarning {
		accent = cfg.ColorWarning
	} else if sev == domain.SeverityDanger {
		accent = cfg.ColorDanger
	}
	txt = RGBToARGB(cfg.ColorText, 255)
	transparent = cfg.TrayTransparent
	if sev != domain.SeverityNormal {
		if cfg.TrayChangeBg {
			bg = RGBToARGB(accent, 255)
			transparent = false
		} else {
			bg = ColorBgCard
			txt = RGBToARGB(accent, 255)
		}
	} else {
		bg = RGBToARGB(cfg.ColorBg, 255)
	}
	border = txt
	return bg, txt, border, transparent
}

func compositeTextMask(dst *Canvas, text, font string, fontPx int, txtARGB uint32, antialias bool) {
	mask := NewCanvas(dst.Width, dst.Height)
	defer mask.Close()
	mask.ClearOpaque(0xFF000000)

	rect := RECT{Left: 0, Top: 0, Right: int32(dst.Width), Bottom: int32(dst.Height)}
	mask.DrawText(text, font, fontPx, true, 0xFFFFFFFF, rect, dtCenter|dtVCenter|dtSingleLine)
	tr := byte(txtARGB >> 16)
	tg := byte(txtARGB >> 8)
	tb := byte(txtARGB)

	for i := 0; i+3 < len(dst.pixels); i += 4 {
		lum := mask.pixels[i+2]
		if mask.pixels[i+1] > lum {
			lum = mask.pixels[i+1]
		}
		if mask.pixels[i] > lum {
			lum = mask.pixels[i]
		}
		if lum == 0 {
			continue
		}
		if !antialias && lum < 110 {
			continue
		}
		if !antialias {
			lum = 255
		}
		a := uint32(lum)
		inv := 255 - a
		dst.pixels[i] = byte((uint32(tb)*a + uint32(dst.pixels[i])*inv) / 255)
		dst.pixels[i+1] = byte((uint32(tg)*a + uint32(dst.pixels[i+1])*inv) / 255)
		dst.pixels[i+2] = byte((uint32(tr)*a + uint32(dst.pixels[i+2])*inv) / 255)
		if dst.pixels[i+3] < lum {
			dst.pixels[i+3] = lum
		}
	}
}

func renderTrayCanvas(percent int, cfg domain.Config, size int) *Canvas {
	if size < 16 {
		size = 16
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 99 {
		percent = 99
	}
	c := NewCanvas(size, size)
	bg, txt, border, transparent := resolveTrayColors(percent, cfg)
	radius := 0
	if cfg.TrayRoundCorners {
		radius = max(4, size/4)
	}
	if !transparent {
		c.FillRoundRect(0, 0, size, size, radius, bg)
	}
	if cfg.TrayShowBorder {
		c.StrokeRoundRect(0, 0, size, size, radius, 1.2, border)
	}
	fontPx := max(10, int(float64(size)*0.58))
	if cfg.TrayFontSize > 0 {
		scaled := int(float64(size) * (float64(cfg.TrayFontSize) / 16.0))
		if scaled >= 9 && scaled <= size-2 {
			fontPx = scaled
		}
	}
	compositeTextMask(c, fmt.Sprintf("%d", percent), cfg.TrayFontName, fontPx, txt, cfg.TrayAntialias)
	return c
}

// RenderTrayIconNRGBA gera imagem RGBA do icone da bandeja para testes e QA.
func RenderTrayIconNRGBA(percent int, cfg domain.Config, size int) *image.NRGBA {
	c := renderTrayCanvas(percent, cfg, size)
	defer c.Close()
	return c.ToNRGBA(false)
}

// CreateTrayHICON cria um HICON 32bpp com transparencia e libera bitmaps temporarios.
func CreateTrayHICON(percent int, cfg domain.Config, size int) (uintptr, error) {
	c := renderTrayCanvas(percent, cfg, size)
	defer c.Close()

	maskBmp, _, _ := procCreateBitmap.Call(uintptr(size), uintptr(size), 1, 1, 0)
	if maskBmp == 0 {
		return 0, fmt.Errorf("CreateBitmap falhou")
	}
	defer procDeleteObject.Call(maskBmp)

	ii := ICONINFO{FIcon: 1, HbmMask: maskBmp, HbmColor: c.HBitmap}
	hIcon, _, err := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	if hIcon == 0 {
		return 0, fmt.Errorf("CreateIconIndirect falhou: %w", err)
	}
	return hIcon, nil
}

// RenderTrayStripPNG exporta tira visual com icones em 48%, 72% e 92% em alta resolucao.
func RenderTrayStripPNG(outPath string, cfg domain.Config) error {
	strip := NewCanvas(240, 88)
	defer strip.Close()
	strip.ClearOpaque(ColorBgBase)

	samples := []struct {
		pct int
		x   int
	}{
		{48, 22},
		{72, 96},
		{92, 170},
	}
	for _, s := range samples {
		ic := renderTrayCanvas(s.pct, cfg, 48)
		for py := 0; py < 48; py++ {
			for px := 0; px < 48; px++ {
				si := (py*48 + px) * 4
				a := float64(ic.pixels[si+3]) / 255.0
				if a > 0 {
					argb := (uint32(ic.pixels[si+3]) << 24) | (uint32(ic.pixels[si+2]) << 16) | (uint32(ic.pixels[si+1]) << 8) | uint32(ic.pixels[si])
					blendPixel(strip.pixels, ((py+12)*strip.Width+(px+s.x))*4, argb, 1.0)
				}
			}
		}
		ic.Close()
		strip.DrawText(fmt.Sprintf("%d%%", s.pct), FontUI, 12, false, ColorTextMuted, RECT{
			Left: int32(s.x), Top: 64, Right: int32(s.x + 48), Bottom: 82,
		}, dtCenter|dtVCenter|dtSingleLine)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, strip.ToNRGBA(true))
}
