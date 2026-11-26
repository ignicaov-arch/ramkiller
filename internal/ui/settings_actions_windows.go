//go:build windows

package ui

import (
	"github.com/ignicaov-arch/ramkiller/internal/domain"
	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

func (a *AppController) applySettingsMutation(hwnd uintptr, id string, cfg *domain.Config) {
	switch id {
	case "chk_always_top":
		cfg.AlwaysOnTop = !cfg.AlwaysOnTop
	case "chk_autorun":
		cfg.LoadOnStartup = !cfg.LoadOnStartup
	case "chk_self_heal":
		cfg.SelfHeal = !cfg.SelfHeal
	case "chk_minimized":
		cfg.StartMinimized = !cfg.StartMinimized
	case "chk_confirm":
		cfg.ConfirmClean = !cfg.ConfirmClean
	case "chk_skip_uac":
		if sys.IsElevated() {
			cfg.SkipUAC = !cfg.SkipUAC
		} else if err := sys.RelaunchAsAdmin("--minimized"); err == nil {
			a.shutdownApp(false)
		}
	case "chk_kill_bg":
		cfg.KillBackgroundApps = !cfg.KillBackgroundApps
	case "chk_auto_pct":
		cfg.AutoCleanThresholdEnabled = !cfg.AutoCleanThresholdEnabled
	case "auto_pct_dec":
		cfg.AutoCleanThresholdPct -= 5
	case "auto_pct_inc":
		cfg.AutoCleanThresholdPct += 5
	case "chk_auto_min":
		cfg.AutoCleanIntervalEnabled = !cfg.AutoCleanIntervalEnabled
	case "auto_min_dec":
		cfg.AutoCleanIntervalMin = stepIntervalMin(cfg.AutoCleanIntervalMin, -1)
	case "auto_min_inc":
		cfg.AutoCleanIntervalMin = stepIntervalMin(cfg.AutoCleanIntervalMin, 1)
	case "anom_mb_dec":
		cfg.AnomalyLimitMB -= 256
	case "anom_mb_inc":
		cfg.AnomalyLimitMB += 256
	case "chk_hotkey":
		cfg.HotkeyEnabled = !cfg.HotkeyEnabled
	case "cycle_hotkey":
		cfg.HotkeyModifiers, cfg.HotkeyKey = nextHotkeyPreset(cfg.HotkeyModifiers, cfg.HotkeyKey)
	case "chk_tray_transp":
		cfg.TrayTransparent = !cfg.TrayTransparent
	case "chk_tray_border":
		cfg.TrayShowBorder = !cfg.TrayShowBorder
	case "chk_tray_round":
		cfg.TrayRoundCorners = !cfg.TrayRoundCorners
	case "chk_tray_chgbg":
		cfg.TrayChangeBg = !cfg.TrayChangeBg
	case "chk_tray_aa":
		cfg.TrayAntialias = !cfg.TrayAntialias
	case "font_sz_dec":
		cfg.TrayFontSize--
	case "font_sz_inc":
		cfg.TrayFontSize++
	case "clr_text":
		if c, ok := ChooseRGBColor(hwnd, cfg.ColorText); ok {
			cfg.ColorText = c
		}
	case "clr_bg":
		if c, ok := ChooseRGBColor(hwnd, cfg.ColorBg); ok {
			cfg.ColorBg = c
		}
	case "clr_warn":
		if c, ok := ChooseRGBColor(hwnd, cfg.ColorWarning); ok {
			cfg.ColorWarning = c
		}
	case "clr_dang":
		if c, ok := ChooseRGBColor(hwnd, cfg.ColorDanger); ok {
			cfg.ColorDanger = c
		}
	case "warn_lvl_dec":
		cfg.WarningLevelPct -= 5
	case "warn_lvl_inc":
		cfg.WarningLevelPct += 5
	case "dang_lvl_dec":
		cfg.DangerLevelPct -= 5
		if cfg.DangerLevelPct < cfg.WarningLevelPct+5 && cfg.WarningLevelPct > 10 {
			cfg.WarningLevelPct = cfg.DangerLevelPct - 5
		}
	case "dang_lvl_inc":
		cfg.DangerLevelPct += 5
	case "tray_single_act":
		cfg.TraySingleClick = (cfg.TraySingleClick + 1) % 3
	case "tray_middle_act":
		cfg.TrayMiddleClick = (cfg.TrayMiddleClick + 1) % 3
	case "chk_balloon":
		cfg.ShowBalloonResult = !cfg.ShowBalloonResult
	case "chk_sound":
		cfg.NotificationSound = !cfg.NotificationSound
	case "chk_log":
		cfg.LogResults = !cfg.LogResults
	}
}

func stepIntervalMin(cur int, dir int) int {
	if dir < 0 {
		if cur <= 5 {
			return 1
		}
		return cur - 5
	}
	if cur < 5 {
		return 5
	}
	return cur + 5
}

func nextHotkeyPreset(mods, vk uint32) (uint32, uint32) {
	switch {
	case mods == (0x0002|0x0004) && vk == 0x70:
		return 0x0002, 0x70 // Ctrl + F1
	case mods == 0x0002 && vk == 0x70:
		return 0x0002 | 0x0001, 0x70 // Ctrl + Alt + F1
	case mods == (0x0002|0x0001) && vk == 0x70:
		return 0x0002 | 0x0004, 0x78 // Ctrl + Shift + F9
	default:
		return 0x0002 | 0x0004, 0x70 // Ctrl + Shift + F1
	}
}
