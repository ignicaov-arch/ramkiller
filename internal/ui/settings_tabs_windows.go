//go:build windows

package ui

import (
	"fmt"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

func renderTabMemory(c *Canvas, x, w int, cfg domain.Config, hoverID string) []HitRegion {
	var hits []HitRegion
	c.FillRoundRect(x, 16, w, 224, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 16, w, 224, RadiusCard, 1, ColorBorder)
	c.DrawText("Clean Areas · Regiões de Limpeza", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 24, int32(x + w - 14), 44}, dtLeft|dtVCenter|dtSingleLine)

	specs := domain.AllRegionSpecs()
	for i, spec := range specs {
		colX := x + 14
		rowIdx := i
		if i >= 4 {
			colX = x + w/2 + 2
			rowIdx = i - 4
		}
		y := 50 + rowIdx*30
		checked := cfg.CleanMask&spec.Flag == spec.Flag
		box := drawCheckboxShort(c, fmt.Sprintf("reg_%d", i), colX, y, w/2-18, checked, spec.Label, hoverID)
		box.Arg = uint32(spec.Flag)
		hits = append(hits, box)
	}
	hits = append(hits, drawCheckbox(c, "chk_kill_bg", x+14, 178, cfg.KillBackgroundApps, "Matar 2º plano ocioso (preserva abertos e núcleo)", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_allow_standby_auto", x+14, 206, cfg.AllowStandbyInAuto, "Permitir Lista de Espera* na limpeza automática", hoverID))

	c.FillRoundRect(x, 250, w, 162, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 250, w, 162, RadiusCard, 1, ColorBorder)
	c.DrawText("Automação, Teto Anômalo e Atalho", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 258, int32(x + w - 14), 278}, dtLeft|dtVCenter|dtSingleLine)

	hits = append(hits, drawCheckboxShort(c, "chk_auto_pct", x+14, 284, 220, cfg.AutoCleanThresholdEnabled, "Limpar quando acima de:", hoverID))
	hits = append(hits, drawStepper(c, "auto_pct", x+w-130, 282, cfg.AutoCleanThresholdPct, "%", hoverID)...)

	hits = append(hits, drawCheckboxShort(c, "chk_auto_min", x+14, 314, 220, cfg.AutoCleanIntervalEnabled, "Limpar a cada:", hoverID))
	hits = append(hits, drawStepper(c, "auto_min", x+w-130, 312, cfg.AutoCleanIntervalMin, "m", hoverID)...)

	c.DrawText("Teto de processo anômalo:", FontUI, 12, false, ColorTextPrimary, RECT{int32(x + 14), 344, int32(x + 240), 368}, dtLeft|dtVCenter|dtSingleLine)
	hits = append(hits, drawStepper(c, "anom_mb", x+w-130, 344, cfg.AnomalyLimitMB, "MB", hoverID)...)

	hits = append(hits, drawCheckboxShort(c, "chk_hotkey", x+14, 378, 220, cfg.HotkeyEnabled, "Atalho global:", hoverID))
	hits = append(hits, drawButton(c, "cycle_hotkey", x+w-150, 376, 136, 26, domain.FormatHotkeyLabel(cfg.HotkeyModifiers, cfg.HotkeyKey), ColorBgElevated, ColorBorder, ColorTextPrimary, false, hoverID))
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
	c.FillRoundRect(x, 16, w, 212, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 16, w, 212, RadiusCard, 1, ColorBorder)
	c.DrawText("Estilo do Ícone na Bandeja", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 24, int32(x + w - 14), 44}, dtLeft|dtVCenter|dtSingleLine)

	hits = append(hits, drawCheckbox(c, "chk_tray_transp", x+14, 50, cfg.TrayTransparent, "Usar fundo transparente", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_tray_border", x+14, 78, cfg.TrayShowBorder, "Mostrar contorno no ícone", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_tray_round", x+14, 106, cfg.TrayRoundCorners, "Cantos arredondados", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_tray_chgbg", x+14, 134, cfg.TrayChangeBg, "Alterar cor de fundo por nível (em vez do texto)", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_tray_aa", x+14, 162, cfg.TrayAntialias, "Suavização de fonte (ClearType)", hoverID))

	c.DrawText("Tamanho da fonte na bandeja:", FontUI, 12, false, ColorTextPrimary, RECT{int32(x + 14), 192, int32(x + 250), 216}, dtLeft|dtVCenter|dtSingleLine)
	hits = append(hits, drawStepper(c, "font_sz", x+w-130, 192, cfg.TrayFontSize, "px", hoverID)...)

	c.FillRoundRect(x, 238, w, 174, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 238, w, 174, RadiusCard, 1, ColorBorder)
	c.DrawText("Cores Semânticas (clique para alterar)", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 246, int32(x + w - 14), 266}, dtLeft|dtVCenter|dtSingleLine)

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
		y := 274 + i*33
		c.DrawText(s.label, FontUI, 12, false, ColorTextPrimary, RECT{int32(x + 14), int32(y), int32(x + 220), int32(y + 26)}, dtLeft|dtVCenter|dtSingleLine)
		swX := x + w - 84
		c.FillRoundRect(swX, y+2, 70, 22, 6, RGBToARGB(s.rgb, 255))
		c.StrokeRoundRect(swX, y+2, 70, 22, 6, 1, ColorBorder)
		hits = append(hits, HitRegion{ID: s.id, Rect: RECT{int32(swX), int32(y), int32(swX + 70), int32(y + 26)}})
	}
	return hits
}
