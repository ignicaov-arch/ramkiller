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
		_ = sys.SetAutorun(cfg.LoadOnStartup, cfg.StartMinimized)
	case "chk_minimized":
		cfg.StartMinimized = !cfg.StartMinimized
		if cfg.LoadOnStartup {
			_ = sys.SetAutorun(true, cfg.StartMinimized)
		}
	case "chk_confirm":
		cfg.ConfirmClean = !cfg.ConfirmClean
	case "chk_skip_uac":
		if sys.IsElevated() {
			cfg.SkipUAC = !cfg.SkipUAC
			_ = sys.SetSkipUACTask(cfg.SkipUAC)
		}
	case "chk_allow_standby_auto":
		cfg.AllowStandbyInAuto = !cfg.AllowStandbyInAuto
	case "chk_auto_pct":
		cfg.AutoCleanThresholdEnabled = !cfg.AutoCleanThresholdEnabled
	case "auto_pct_dec":
		cfg.AutoCleanThresholdPct -= 5
	case "auto_pct_inc":
		cfg.AutoCleanThresholdPct += 5
	case "chk_auto_min":
		cfg.AutoCleanIntervalEnabled = !cfg.AutoCleanIntervalEnabled
	case "auto_min_dec":
		cfg.AutoCleanIntervalMin -= 5
	case "auto_min_inc":
		cfg.AutoCleanIntervalMin += 5
	case "chk_hotkey":
		cfg.HotkeyEnabled = !cfg.HotkeyEnabled
	case "cycle_hotkey":
		cfg.HotkeyKey = nextHotkeyVK(cfg.HotkeyKey)
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

func nextHotkeyVK(cur uint32) uint32 {
	switch cur {
	case 0x70:
		return 0x71 // F2
	case 0x71:
		return 0x72 // F3
	case 0x72:
		return 0x78 // F9
	default:
		return 0x70 // F1
	}
}
