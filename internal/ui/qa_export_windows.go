//go:build windows

package ui

import (
	"path/filepath"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

// RenderQAScreenshots gera imagens PNG reais do Painel, das 4 abas e da bandeja para auditoria visual.
func RenderQAScreenshots(outDir string) error {
	cfg := domain.DefaultConfig()
	stats, err := sys.ReadMemoryStats(-1, cfg.AnomalyLimitMB)
	if err != nil {
		stats = domain.MemoryStats{
			Physical:    domain.BankStat{UsedBytes: 8 * 1024 * 1024 * 1024, FreeBytes: 8 * 1024 * 1024 * 1024, TotalBytes: 16 * 1024 * 1024 * 1024, Percent: 50},
			Pagefile:    domain.BankStat{UsedBytes: 9 * 1024 * 1024 * 1024, FreeBytes: 11 * 1024 * 1024 * 1024, TotalBytes: 20 * 1024 * 1024 * 1024, Percent: 45},
			SystemCache: domain.BankStat{UsedBytes: 512 * 1024 * 1024, FreeBytes: 4096 * 1024 * 1024, TotalBytes: 4608 * 1024 * 1024, Percent: 11},
			Elevated:    true,
		}
	}

	mainCanvas, _ := RenderMainCanvas(stats, cfg, "+2.85 GB liberados (Modo Kernel · 11 encerrados · 96 reciclados · 42 ms)", "", false)
	defer mainCanvas.Close()
	if err := mainCanvas.SavePNG(filepath.Join(outDir, "main_window.png")); err != nil {
		return err
	}

	tabNames := []string{"settings_general.png", "settings_memory.png", "settings_appearance.png", "settings_tray.png"}
	for tab, fileName := range tabNames {
		sc, _ := RenderSettingsCanvas(tab, cfg, true, "")
		saveErr := sc.SavePNG(filepath.Join(outDir, fileName))
		sc.Close()
		if saveErr != nil {
			return saveErr
		}
	}

	return RenderTrayStripPNG(filepath.Join(outDir, "tray_icons_strip.png"), cfg)
}
