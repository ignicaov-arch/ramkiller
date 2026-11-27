//go:build windows

package ui

import (
	"fmt"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const (
	mainWinWidth  = 500
	mainWinHeight = 720
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
	const cardH = 62
	const pad = 12
	c.FillRoundRect(x, y, w, cardH, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, y, w, cardH, RadiusCard, 1, ColorBorder)

	accent, soft := severityPalette(bank.Percent, cfg)
	c.DrawText(title, FontUI, 13, true, ColorTextPrimary, RECT{int32(x + pad), int32(y + 7), int32(x + w - 90), int32(y + 25)}, dtLeft|dtVCenter|dtSingleLine)

	badgeW := 48
	c.FillRoundRect(x+w-pad-badgeW, y+7, badgeW, 20, 6, soft)
	c.DrawText(fmt.Sprintf("%d%%", bank.Percent), FontUI, 12, true, accent, RECT{int32(x + w - pad - badgeW), int32(y + 7), int32(x + w - pad), int32(y + 27)}, dtCenter|dtVCenter|dtSingleLine)

	barX, barY, barW, barH := x+pad, y+31, w-pad*2, 6
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
	c.DrawText(sub, FontUI, 11, false, ColorTextMuted, RECT{int32(x + pad), int32(y + 41), int32(x + w - pad), int32(y + 57)}, dtLeft|dtVCenter|dtSingleLine)
}

// RenderMainCanvas desenha o Painel Principal do RamKiller com rolagem completa da lista de processos.
func RenderMainCanvas(stats domain.MemoryStats, cfg domain.Config, lastSummary, hoverID string, cleaning bool, scrollY ...int) (*Canvas, []HitRegion) {
	c := NewCanvas(mainWinWidth, mainWinHeight)
	c.ClearOpaque(ColorBgBase)
	var hits []HitRegion

	curScroll := 0
	if len(scrollY) > 0 {
		curScroll = scrollY[0]
	}

	const pad = 16
	const contentW = mainWinWidth - pad*2

	c.DrawText("RamKiller", FontUI, 18, true, ColorTextPrimary, RECT{pad, 14, 134, 40}, dtLeft|dtVCenter|dtSingleLine)
	if stats.Elevated {
		c.FillRoundRect(136, 16, 146, 24, 6, ColorGreenSoft)
		c.DrawText("Modo Kernel (Admin)", FontUI, 11, true, ColorGreen, RECT{136, 16, 282, 40}, dtCenter|dtVCenter|dtSingleLine)
	} else {
		hits = append(hits, drawButton(c, "elevate_admin", 136, 15, 158, 26, "Modo Usuário · Elevar", ColorAmberSoft, ColorAmber, ColorAmber, true, hoverID))
	}
	hits = append(hits, drawButton(c, "open_settings", mainWinWidth-pad-106, 15, 106, 26, "Configurações", ColorBgElevated, ColorBorder, ColorTextPrimary, false, hoverID))

	drawMemoryBankCard(c, pad, 50, contentW, "Memória Física", stats.Physical, cfg)
	drawMemoryBankCard(c, pad, 120, contentW, "Arquivo de Paginação (Pagefile)", stats.Pagefile, cfg)
	drawMemoryBankCard(c, pad, 190, contentW, "Cache do Sistema", stats.SystemCache, cfg)

	procHits, _ := drawProcessesCardFull(c, pad, contentW, stats.TopProcesses, curScroll, hoverID)
	hits = append(hits, procHits...)

	statusTxt := lastSummary
	if cleaning {
		statusTxt = "Limpando RAM, Paginação, 2º plano e Kernel..."
	} else if statusTxt == "" {
		statusTxt = "Pronto · 1 clique limpa RAM, Paginação, 2º plano e Kernel."
	}
	c.FillRoundRect(pad, 582, contentW, 30, 8, ColorBgCard)
	c.StrokeRoundRect(pad, 582, contentW, 30, 8, 1, ColorBorder)
	c.DrawText(statusTxt, FontUI, 11, false, ColorTextMuted, RECT{pad + 10, 582, mainWinWidth - pad - 10, 612}, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)

	hits = append(hits, drawButton(c, "clean_now", pad, 622, contentW, 42, "Limpar Tudo Agora", ColorGreen, 0, ColorBgBase, true, hoverID))
	hits = append(hits, drawButton(c, "stop_until_reboot", pad, 672, contentW, 36, "Encerrar até Reiniciar o PC", ColorRedSoft, ColorRed, ColorRed, true, hoverID))

	return c, hits
}
