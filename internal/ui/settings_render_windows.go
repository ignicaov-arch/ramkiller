//go:build windows

package ui

import "github.com/ignicaov-arch/ramkiller/internal/domain"

const (
	settingsWinWidth  = 680
	settingsWinHeight = 480
	sidebarWidth      = 148
)

var settingsTabTitles = []string{"Geral", "Limpeza", "Aparência", "Bandeja"}

// RenderSettingsCanvas desenha a janela de configuracoes com 4 abas e cantos concentricos.
func RenderSettingsCanvas(tab int, cfg domain.Config, elevated bool, hoverID string) (*Canvas, []HitRegion) {
	c := NewCanvas(settingsWinWidth, settingsWinHeight)
	c.ClearOpaque(ColorBgBase)
	var hits []HitRegion

	c.FillRect(0, 0, sidebarWidth, settingsWinHeight, ColorBgCard)
	c.FillRect(sidebarWidth, 0, 1, settingsWinHeight, ColorBorder)
	c.DrawText("Definições", FontUI, 15, true, ColorTextPrimary, RECT{16, 16, sidebarWidth - 12, 40}, dtLeft|dtVCenter|dtSingleLine)

	for i, name := range settingsTabTitles {
		y := 52 + i*38
		id := tabID(i)
		bg := uint32(0)
		fg := ColorTextMuted
		bold := false
		if i == tab {
			bg = ColorGreenSoft
			fg = ColorGreen
			bold = true
		} else if hoverID == id {
			bg = ColorWhiteSoft
			fg = ColorTextPrimary
		}
		if bg != 0 {
			c.FillRoundRect(10, y, sidebarWidth-20, 32, 8, bg)
		}
		c.DrawText(name, FontUI, 13, bold, fg, RECT{22, int32(y), sidebarWidth - 16, int32(y + 32)}, dtLeft|dtVCenter|dtSingleLine)
		hits = append(hits, HitRegion{ID: id, Rect: RECT{10, int32(y), sidebarWidth - 10, int32(y + 32)}})
	}

	cx := sidebarWidth + 18
	cw := settingsWinWidth - cx - 18
	switch tab {
	case 0:
		hits = append(hits, renderTabGeneral(c, cx, cw, cfg, elevated, hoverID)...)
	case 1:
		hits = append(hits, renderTabMemory(c, cx, cw, cfg, hoverID)...)
	case 2:
		hits = append(hits, renderTabAppearance(c, cx, cw, cfg, hoverID)...)
	case 3:
		hits = append(hits, renderTabTray(c, cx, cw, cfg, hoverID)...)
	}

	c.FillRect(sidebarWidth+1, settingsWinHeight-54, settingsWinWidth-sidebarWidth-1, 1, ColorBorder)
	hits = append(hits, drawButton(c, "settings_reset", cx, settingsWinHeight-42, 136, 30, "Restaurar Padrão", ColorBgElevated, ColorBorder, ColorTextPrimary, false, hoverID))
	hits = append(hits, drawButton(c, "settings_close", settingsWinWidth-18-96, settingsWinHeight-42, 96, 30, "Fechar", ColorGreen, 0, ColorBgBase, true, hoverID))
	return c, hits
}

func tabID(i int) string {
	switch i {
	case 0:
		return "tab_0"
	case 1:
		return "tab_1"
	case 2:
		return "tab_2"
	default:
		return "tab_3"
	}
}

func renderTabGeneral(c *Canvas, x, w int, cfg domain.Config, elevated bool, hoverID string) []HitRegion {
	var hits []HitRegion
	c.FillRoundRect(x, 16, w, 232, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 16, w, 232, RadiusCard, 1, ColorBorder)
	c.DrawText("Comportamento Geral e Autoregeneração", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 24, int32(x + w - 14), 44}, dtLeft|dtVCenter|dtSingleLine)

	hits = append(hits, drawCheckbox(c, "chk_always_top", x+14, 52, cfg.AlwaysOnTop, "Manter janela sempre no topo", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_autorun", x+14, 82, cfg.LoadOnStartup, "Iniciar automaticamente ao ligar o PC", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_self_heal", x+14, 112, cfg.SelfHeal, "Autoregenerar se encerrado por fora (guardião)", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_minimized", x+14, 142, cfg.StartMinimized, "Iniciar minimizado na bandeja", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_confirm", x+14, 172, cfg.ConfirmClean, "Confirmar antes de iniciar limpeza manual", hoverID))

	uacLabel := "Ignorar aviso UAC ao ligar (Tarefa Agendada Admin)"
	if !elevated {
		uacLabel = "Ignorar aviso UAC ao ligar (clique para elevar Admin)"
	}
	hits = append(hits, drawCheckbox(c, "chk_skip_uac", x+14, 202, cfg.SkipUAC, uacLabel, hoverID))

	c.FillRoundRect(x, 260, w, 150, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 260, w, 150, RadiusCard, 1, ColorBorder)
	c.DrawText("Arquitetura de Proteção", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 270, int32(x + w - 14), 290}, dtLeft|dtVCenter|dtSingleLine)
	c.DrawText("• Janelas Abertas: detectadas via WinSta0\\Default e preservadas.", FontUI, 12, false, ColorTextMuted, RECT{int32(x + 14), 296, int32(x + w - 14), 318}, dtLeft|dtVCenter|dtSingleLine)
	c.DrawText("• Antigravity + language_server: hub ativo preservado e reciclado.", FontUI, 12, false, ColorTextMuted, RECT{int32(x + 14), 322, int32(x + w - 14), 344}, dtLeft|dtVCenter|dtSingleLine)
	c.DrawText("• Modo Kernel (Admin): limpa todas as 8 áreas e encerra 2º plano.", FontUI, 12, false, ColorTextMuted, RECT{int32(x + 14), 348, int32(x + w - 14), 370}, dtLeft|dtVCenter|dtSingleLine)
	c.DrawText("• Encerrar até Reiniciar: pausa guardião só até o próximo boot.", FontUI, 12, false, ColorTextMuted, RECT{int32(x + 14), 374, int32(x + w - 14), 396}, dtLeft|dtVCenter|dtSingleLine)
	return hits
}

func trayActionName(a domain.TrayAction) string {
	switch a {
	case domain.ActionCleanMemory:
		return "Limpar Memória"
	case domain.ActionOpenTaskMgr:
		return "Gerenciador de Tarefas"
	default:
		return "Abrir / Ocultar Painel"
	}
}

func renderTabTray(c *Canvas, x, w int, cfg domain.Config, hoverID string) []HitRegion {
	var hits []HitRegion
	c.FillRoundRect(x, 16, w, 108, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 16, w, 108, RadiusCard, 1, ColorBorder)
	c.DrawText("Limiares de Alerta Visual", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 24, int32(x + w - 14), 44}, dtLeft|dtVCenter|dtSingleLine)

	c.DrawText("Nível de Alerta inicia em:", FontUI, 13, false, ColorTextPrimary, RECT{int32(x + 14), 52, int32(x + 250), 76}, dtLeft|dtVCenter|dtSingleLine)
	hits = append(hits, drawStepper(c, "warn_lvl", x+w-130, 52, cfg.WarningLevelPct, "%", hoverID)...)
	c.DrawText("Nível de Perigo inicia em:", FontUI, 13, false, ColorTextPrimary, RECT{int32(x + 14), 84, int32(x + 250), 108}, dtLeft|dtVCenter|dtSingleLine)
	hits = append(hits, drawStepper(c, "dang_lvl", x+w-130, 84, cfg.DangerLevelPct, "%", hoverID)...)

	c.FillRoundRect(x, 136, w, 108, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 136, w, 108, RadiusCard, 1, ColorBorder)
	c.DrawText("Controle do Mouse na Bandeja", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 144, int32(x + w - 14), 164}, dtLeft|dtVCenter|dtSingleLine)
	c.DrawText("Clique único esquerdo:", FontUI, 13, false, ColorTextPrimary, RECT{int32(x + 14), 172, int32(x + 200), 198}, dtLeft|dtVCenter|dtSingleLine)
	hits = append(hits, drawButton(c, "tray_single_act", x+w-190, 171, 176, 26, trayActionName(cfg.TraySingleClick), ColorBgElevated, ColorBorder, ColorTextPrimary, false, hoverID))
	c.DrawText("Clique central (scroll):", FontUI, 13, false, ColorTextPrimary, RECT{int32(x + 14), 206, int32(x + 200), 232}, dtLeft|dtVCenter|dtSingleLine)
	hits = append(hits, drawButton(c, "tray_middle_act", x+w-190, 205, 176, 26, trayActionName(cfg.TrayMiddleClick), ColorBgElevated, ColorBorder, ColorTextPrimary, false, hoverID))

	c.FillRoundRect(x, 256, w, 124, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, 256, w, 124, RadiusCard, 1, ColorBorder)
	c.DrawText("Notificações e Auditoria", FontUI, 13, true, ColorTextPrimary, RECT{int32(x + 14), 264, int32(x + w - 14), 284}, dtLeft|dtVCenter|dtSingleLine)
	hits = append(hits, drawCheckbox(c, "chk_balloon", x+14, 290, cfg.ShowBalloonResult, "Exibir balão com memória liberada", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_sound", x+14, 318, cfg.NotificationSound, "Ativar som na notificação", hoverID))
	hits = append(hits, drawCheckbox(c, "chk_log", x+14, 346, cfg.LogResults, "Gravar histórico em ramkiller.log", hoverID))
	return hits
}
