//go:build windows

package ui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

func TestTrayIconCreationAndCleanup(t *testing.T) {
	cfg := domain.DefaultConfig()
	for _, pct := range []int{42, 68, 94} {
		img := RenderTrayIconNRGBA(pct, cfg, 32)
		if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
			t.Fatalf("dimensao invalida para %d%%: %v", pct, img.Bounds())
		}
		hIcon, err := CreateTrayHICON(pct, cfg, 32)
		if err != nil || hIcon == 0 {
			t.Fatalf("CreateTrayHICON(%d) falhou: %v", pct, err)
		}
		procDestroyIcon.Call(hIcon)
	}
}

func TestRenderQAScreenshotsValidPNGs(t *testing.T) {
	outDir := t.TempDir()
	if err := RenderQAScreenshots(outDir); err != nil {
		t.Fatalf("RenderQAScreenshots falhou: %v", err)
	}
	expected := map[string][2]int{
		"main_window.png":         {mainWinWidth, mainWinHeight},
		"settings_general.png":    {settingsWinWidth, settingsWinHeight},
		"settings_memory.png":     {settingsWinWidth, settingsWinHeight},
		"settings_appearance.png": {settingsWinWidth, settingsWinHeight},
		"settings_tray.png":       {settingsWinWidth, settingsWinHeight},
		"tray_icons_strip.png":    {240, 88},
	}
	for name, dims := range expected {
		f, err := os.Open(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("arquivo esperado ausente %s: %v", name, err)
		}
		cfgImg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Fatalf("PNG invalido %s: %v", name, err)
		}
		if cfgImg.Width != dims[0] || cfgImg.Height != dims[1] {
			t.Errorf("%s dimensoes %dx%d != esperado %dx%d", name, cfgImg.Width, cfgImg.Height, dims[0], dims[1])
		}
	}
}
