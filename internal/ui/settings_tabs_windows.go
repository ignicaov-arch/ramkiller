//go:build windows

package ui

import (
	"fmt"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

func renderTabMemory(c *Canvas, x, w int, cfg domain.Config, hoverID string) []HitRegion {
	var hits []HitRegion
	c.FillRoundRect(x, 16, w, 206, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 16, w, 206, RadiusCard, 1, ColorBorder)
	c.DrawText("Regiões de Memória (Limpeza Profunda)", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 24, int32(x + w - 14), 44}, dtLeft|dtVCenter|dtSingleLine)

	specs := domain.AllRegionSpecs()
	for i, spec := range specs {
		colX := x + 14
		rowIdx := i
		if i >= 4 {
			colX = x + w/2 + 2
			rowIdx = i - 4
		}
		y := 52 + rowIdx*32
		checked := cfg.CleanMask&spec.Flag == spec.Flag
		id := fmt.Sprintf("reg_%d", i)
		box := drawCheckboxShort(c, id, colX, y, w/2-18, checked, spec.Label, hoverID)
		box.Arg = uint32(spec.Flag)
		hits = append(hits, box)
	}
	hits = append(hits, drawCheckbox(c, "chk_allow_standby_auto", x+14, 188, cfg.AllowStandbyInAuto, "Permitir Lista de Espera* na limpeza automática", hoverID))

	c.FillRoundRect(x, 232, w, 150, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 232, w, 150, RadiusCard, 1, ColorBorder)
	c.DrawText("Automação e Atalho Global", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 240, int32(x + w - 14), 260}, dtLeft|dtVCenter|dtSingleLine)

	hits = append(hits, drawCheckboxShort(c, "chk_auto_pct", x+14, 268, 220, cfg.AutoCleanThresholdEnabled, "Limpar quando acima de:", hoverID))
	hits = append(hits, drawStepper(c, "auto_pct", x+w-130, 266, cfg.AutoCleanThresholdPct, "%", hoverID)...)

	hits = append(hits, drawCheckboxShort(c, "chk_auto_min", x+14, 300, 220, cfg.AutoCleanIntervalEnabled, "Limpar a cada:", hoverID))
	hits = append(hits, drawStepper(c, "auto_min", x+w-130, 298, cfg.AutoCleanIntervalMin, "m", hoverID)...)

	hits = append(hits, drawCheckboxShort(c, "chk_hotkey", x+14, 336, 220, cfg.HotkeyEnabled, "Atalho global:", hoverID))
	hits = append(hits, drawButton(c, "cycle_hotkey", x+w-150, 334, 136, 26, domain.FormatHotkeyLabel(cfg.HotkeyModifiers, cfg.HotkeyKey), ColorBgElevated, ColorBorder, ColorTextPrimary, false, hoverID))
	return hits
}

func drawCheckboxShort(c *Canvas, id string, x, y, maxW int, checked bool, label string, hoverID string) HitRegion {
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
	c.DrawText(label, FontUI, 11, false, ColorTextPrimary, RECT{int32(x + 22), int32(y), int32(x + maxW), int32(y + 22)}, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	return HitRegion{ID: id, Rect: RECT{int32(x), int32(y), int32(x + maxW), int32(y + 22)}}
}

func renderTabAppearance(c *Canvas, x, w int, cfg domain.Config, hoverID string) []HitRegion {
	var hits []HitRegion
	c.FillRoundRect(x, 16, w, 178, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 16, w, 178, RadiusCard, 1, ColorBorder)
	c.DrawText("Estilo do Ícone na Bandeja", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 24, int32(x + w - 14), 44}, dtLeft|dtVCenter|dtSingleLine)

	hits = append(hits, drawCheckbox(c, "chk_tray_transp", x+14, 52, cfg.TrayTransparent, "Usar fundo transparente", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_tray_border", x+14, 80, cfg.TrayShowBorder, "Mostrar contorno no ícone", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_tray_round", x+14, 108, cfg.TrayRoundCorners, "Cantos arredondados", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_tray_chgbg", x+14, 136, cfg.TrayChangeBg, "Alterar cor de fundo por nível (em vez do texto)", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_tray_aa", x+14, 164, cfg.TrayAntialias, "Suavização de fonte (ClearType)", hoverID))

	c.FillRoundRect(x, 206, w, 176, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 206, w, 176, RadiusCard, 1, ColorBorder)
	c.DrawText("Cores Semânticas (clique para alterar)", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 214, int32(x + w - 14), 234}, dtLeft|dtVCenter|dtSingleLine)

	swatches := []struct {
		id    string
		label string
		rgb   uint32
	}{
		{"clr_text", "Cor do Texto", cfg.ColorText},
		{"clr_bg", "Fundo Normal", cfg.ColorBg},
		{"clr_warn", "Nível de Alerta", cfg.ColorWarning},
		{"clr_dang", "Nível de Perigo", cfg.ColorDanger},
	}
	for i, s := range swatches {
		y := 244 + i*33
		c.DrawText(s.label, FontUI, 12, false, ColorTextPrimary, RECT{int32(x + 14), int32(y), int32(x + 220), int32(y + 26)}, dtLeft|dtVCenter|dtSingleLine)
		swX := x + w - 84
		c.FillRoundRect(swX, y+2, 70, 22, 6, RGBToARGB(s.rgb, 255))
		c.StrokeRoundRect(swX, y+2, 70, 22, 6, 1, ColorBorder)
		hits = append(hits, HitRegion{ID: s.id, Rect: RECT{int32(swX), int32(y), int32(swX + 70), int32(y + 26)}})
	}
	return hits
}
