//go:build windows

package ui

import (
	"os"
	"strings"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

func (a *AppController) handleMainClick(hit HitRegion) {
	switch {
	case hit.ID == "clean_now":
		a.triggerClean(domain.SourceManual, 0)
	case hit.ID == "clean_deep":
		a.triggerClean(domain.SourceManual, domain.MaskAll)
	case hit.ID == "open_settings":
		a.openSettingsWindow()
	case hit.ID == "elevate_admin":
		if err := sys.RelaunchAsAdmin("--minimized"); err == nil {
			procPostMessageW.Call(a.mainHWND, WM_DESTROY, 0, 0)
		}
	case strings.HasPrefix(hit.ID, "trim_pid_") && hit.Arg > 0:
		if err := sys.TrimProcessWorkingSet(hit.Arg); err == nil {
			a.lastSummary = "Working Set do processo reciclado com sucesso."
		} else {
			a.lastSummary = "Sem permissão para reciclar este processo (tente Modo Kernel)."
		}
		a.refreshStats()
		procInvalidateRect.Call(a.mainHWND, 0, 0)
	}
}

func (a *AppController) handleTrayEvent(ev uint32) {
	switch ev {
	case WM_LBUTTONUP:
		a.executeTrayAction(a.getConfig().TraySingleClick)
	case WM_MBUTTONUP:
		a.executeTrayAction(a.getConfig().TrayMiddleClick)
	case WM_RBUTTONUP:
		cmd := ShowTrayContextMenu(a.mainHWND, a.getConfig())
		a.handleTrayMenuCommand(cmd)
	}
}

func (a *AppController) executeTrayAction(act domain.TrayAction) {
	switch act {
	case domain.ActionCleanMemory:
		a.triggerClean(domain.SourceManual, 0)
	case domain.ActionOpenTaskMgr:
		_ = sys.OpenTaskManager()
	default:
		a.toggleMainWindow()
	}
}

func (a *AppController) handleTrayMenuCommand(cmd uint32) {
	switch {
	case cmd == cmdTrayToggle:
		a.toggleMainWindow()
	case cmd == cmdTrayClean:
		a.triggerClean(domain.SourceManual, 0)
	case cmd == cmdTrayDeep:
		a.triggerClean(domain.SourceManual, domain.MaskAll)
	case cmd == cmdTraySettings:
		a.openSettingsWindow()
	case cmd == cmdTrayExit:
		procPostMessageW.Call(a.mainHWND, WM_DESTROY, 0, 0)
		os.Exit(0)
	case cmd >= cmdRegionBase && cmd < cmdRegionBase+8:
		specs := domain.AllRegionSpecs()
		cfg := a.getConfig()
		cfg.CleanMask ^= specs[cmd-cmdRegionBase].Flag
		a.applyNewConfig(cfg)
	case cmd == cmdThreshOff:
		cfg := a.getConfig()
		cfg.AutoCleanThresholdEnabled = false
		a.applyNewConfig(cfg)
	case cmd >= cmdThreshBase && cmd < cmdThreshBase+uint32(len(thresholdOptions)):
		cfg := a.getConfig()
		cfg.AutoCleanThresholdEnabled = true
		cfg.AutoCleanThresholdPct = thresholdOptions[cmd-cmdThreshBase]
		a.applyNewConfig(cfg)
	case cmd == cmdIntervalOff:
		cfg := a.getConfig()
		cfg.AutoCleanIntervalEnabled = false
		a.applyNewConfig(cfg)
	case cmd >= cmdIntervalBase && cmd < cmdIntervalBase+uint32(len(intervalOptions)):
		cfg := a.getConfig()
		cfg.AutoCleanIntervalEnabled = true
		cfg.AutoCleanIntervalMin = intervalOptions[cmd-cmdIntervalBase]
		a.applyNewConfig(cfg)
	}
}
