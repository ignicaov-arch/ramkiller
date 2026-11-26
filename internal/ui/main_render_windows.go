//go:build windows

package ui

import (
	"fmt"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const (
	mainWinWidth  = 460
	mainWinHeight = 592
)

func severityPalette(pct int, cfg domain.Config) (uint32, uint32) {
	switch domain.SeverityForPercent(pct, cfg) {
	case domain.SeverityDanger:
		return ColorRed, ColorRedSoft
	case domain.SeverityWarning:
		return ColorAmber, ColorAmberSoft
	default:
		return ColorGreen, ColorGreenSoft
	}
}

func drawMemoryBankCard(c *Canvas, x, y, w int, title string, bank domain.BankStat, cfg domain.Config) {
	const cardH = 66
	const pad = 12
	c.FillRoundRect(x, y, w, cardH, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, y, w, cardH, RadiusCard, 1, ColorBorder)

	accent, soft := severityPalette(bank.Percent, cfg)
	c.DrawText(title, FontUI, 13, true, ColorTextPrimary, RECT{int32(x + pad), int32(y + 8), int32(x + w - 90), int32(y + 26)}, dtLeft|dtVCenter|dtSingleLine)

	badgeW := 48
	c.FillRoundRect(x+w-pad-badgeW, y+8, badgeW, 20, 6, soft)
	c.DrawText(fmt.Sprintf("%d%%", bank.Percent), FontUI, 12, true, accent, RECT{int32(x + w - pad - badgeW), int32(y + 8), int32(x + w - pad), int32(y + 28)}, dtCenter|dtVCenter|dtSingleLine)

	barX, barY, barW, barH := x+pad, y+34, w-pad*2, 6
	c.FillRoundRect(barX, barY, barW, barH, 3, ColorBgBase)
	fillW := (barW * bank.Percent) / 100
	if fillW < 6 && bank.Percent > 0 {
		fillW = 6
	}
	if fillW > 0 {
		c.FillRoundRect(barX, barY, fillW, barH, 3, accent)
	}

	sub := fmt.Sprintf("Usada: %s   ·   Livre: %s   ·   Total: %s",
		domain.FormatBytes(bank.UsedBytes),
		domain.FormatBytes(bank.FreeBytes),
		domain.FormatBytes(bank.TotalBytes),
	)
	c.DrawText(sub, FontUI, 11, false, ColorTextMuted, RECT{int32(x + pad), int32(y + 44), int32(x + w - pad), int32(y + 60)}, dtLeft|dtVCenter|dtSingleLine)
}

func drawProcessesCard(c *Canvas, x, y, w int, procs []domain.ProcessInfo, hoverID string) []HitRegion {
	const cardH = 198
	const pad = 12
	var hits []HitRegion

	c.FillRoundRect(x, y, w, cardH, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, y, w, cardH, RadiusCard, 1, ColorBorder)
	c.DrawText("Top 5 Processos na RAM", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + pad), int32(y + 10), int32(x + w - pad), int32(y + 28)}, dtLeft|dtVCenter|dtSingleLine)

	if len(procs) == 0 {
		c.DrawText("Nenhum processo acima do filtro.", FontUI, 12, false, ColorTextMuted, RECT{int32(x + pad), int32(y + 60), int32(x + w - pad), int32(y + 100)}, dtCenter|dtVCenter|dtSingleLine)
		return hits
	}

	rowY := y + 34
	for i, p := range procs {
		if i >= 5 {
			break
		}
		if i > 0 {
			c.FillRect(x+pad, rowY-2, w-pad*2, 1, ColorBorder)
		}
		nameColor := ColorTextPrimary
		if p.Anomaly {
			nameColor = ColorAmber
		}
		label := fmt.Sprintf("%s  (%d)", p.Name, p.PID)
		c.DrawText(label, FontUI, 12, false, nameColor, RECT{int32(x + pad), int32(rowY), int32(x + w - 164), int32(rowY + 28)}, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)

		memColor := ColorTextMuted
		if p.Anomaly {
			memColor = ColorRed
		}
		c.DrawText(domain.FormatBytes(p.PrivateBytes), FontUI, 12, true, memColor, RECT{int32(x + w - 162), int32(rowY), int32(x + w - 88), int32(rowY + 28)}, dtRight|dtVCenter|dtSingleLine)

		if p.Protected {
			c.FillRoundRect(x+w-pad-68, rowY+4, 68, 20, 6, ColorWhiteSoft)
			c.DrawText("Protegido", FontUI, 11, false, ColorTextMuted, RECT{int32(x + w - pad - 68), int32(rowY + 4), int32(x + w - pad), int32(rowY + 24)}, dtCenter|dtVCenter|dtSingleLine)
		} else {
			btnID := fmt.Sprintf("trim_pid_%d", p.PID)
			h := drawButton(c, btnID, x+w-pad-68, rowY+3, 68, 22, "Reciclar", ColorBgElevated, ColorBorder, ColorTextPrimary, false, hoverID)
			h.Arg = p.PID
			hits = append(hits, h)
		}
		rowY += 32
	}
	return hits
}

// RenderMainCanvas desenha todo o Painel Principal do RamKiller em buffer double-buffered.
func RenderMainCanvas(stats domain.MemoryStats, cfg domain.Config, lastSummary, hoverID string, cleaning bool) (*Canvas, []HitRegion) {
	c := NewCanvas(mainWinWidth, mainWinHeight)
	c.ClearOpaque(ColorBgBase)
	var hits []HitRegion

	const pad = 16
	const contentW = mainWinWidth - pad*2

	// Header
	c.DrawText("RamKiller", FontUI, 18, true, ColorTextPrimary, RECT{pad, 14, 134, 40}, dtLeft|dtVCenter|dtSingleLine)
	if stats.Elevated {
		c.FillRoundRect(136, 16, 146, 24, 6, ColorGreenSoft)
		c.DrawText("Modo Kernel (Admin)", FontUI, 11, true, ColorGreen, RECT{136, 16, 282, 40}, dtCenter|dtVCenter|dtSingleLine)
	} else {
		h := drawButton(c, "elevate_admin", 136, 15, 158, 26, "Modo Usuário · Elevar", ColorAmberSoft, ColorAmber, ColorAmber, true, hoverID)
		hits = append(hits, h)
	}
	hits = append(hits, drawButton(c, "open_settings", mainWinWidth-pad-106, 15, 106, 26, "Configurações", ColorBgElevated, ColorBorder, ColorTextPrimary, false, hoverID))

	// 3 Cards de telemetria
	drawMemoryBankCard(c, pad, 54, contentW, "Memória Física", stats.Physical, cfg)
	drawMemoryBankCard(c, pad, 128, contentW, "Arquivo de Paginação / Commit", stats.Pagefile, cfg)
	drawMemoryBankCard(c, pad, 202, contentW, "Cache do Sistema", stats.SystemCache, cfg)

	// Top 5 Processos
	hits = append(hits, drawProcessesCard(c, pad, 278, contentW, stats.TopProcesses, hoverID)...)

	// Barra de status / resultado da ultima limpeza
	statusTxt := lastSummary
	if cleaning {
		statusTxt = "Executando reciclagem segura de memória..."
	} else if statusTxt == "" {
		statusTxt = "Pronto · Limpeza cirúrgica sem encerrar seus aplicativos."
	}
	c.FillRoundRect(pad, 486, contentW, 30, 8, ColorBgCard)
	c.StrokeRoundRect(pad, 486, contentW, 30, 8, 1, ColorBorder)
	c.DrawText(statusTxt, FontUI, 11, false, ColorTextMuted, RECT{pad + 10, 486, mainWinWidth - pad - 10, 516}, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)

	// Botoes principais de acao
	btnW := (contentW - 10) / 2
	hits = append(hits, drawButton(c, "clean_now", pad, 528, btnW, 44, "Limpar Memória Agora", ColorGreen, 0, ColorBgBase, true, hoverID))
	hits = append(hits, drawButton(c, "clean_deep", pad+btnW+10, 528, btnW, 44, "Limpeza Profunda", ColorBgElevated, ColorBorder, ColorTextPrimary, true, hoverID))

	return c, hits
}
