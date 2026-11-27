//go:build windows

package ui

import (
	"context"
	"time"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

func (a *AppController) refreshStats() {
	cfg := a.getConfig()
	if st, err := sys.ReadMemoryStats(5, cfg.AnomalyLimitMB); err == nil {
		a.stats = st
	}
}

func (a *AppController) forceTrayRefresh() {
	cfg := a.getConfig()
	pct := a.stats.Physical.Percent
	hIcon, err := CreateTrayHICON(pct, cfg, 32)
	if err == nil {
		a.lastIconPct = pct
		a.tray.Update(hIcon, domain.FormatTooltip(a.stats))
	}
}

func (a *AppController) onTick() {
	if sys.IsPausedForCurrentBoot() {
		a.shutdownApp(true)
		return
	}
	a.tickCount++
	a.refreshStats()
	cfg := a.getConfig()
	pct := a.stats.Physical.Percent

	var newIcon uintptr
	if pct != a.lastIconPct {
		if ic, err := CreateTrayHICON(pct, cfg, 32); err == nil {
			newIcon = ic
			a.lastIconPct = pct
		}
	}
	a.tray.Update(newIcon, domain.FormatTooltip(a.stats))

	if cfg.SelfHeal && a.tickCount%3 == 0 && !sys.IsPausedForCurrentBoot() {
		_ = sys.EnsureGuardianRunning()
	}

	if vis, _, _ := procIsWindowVisible.Call(a.mainHWND); vis != 0 {
		procInvalidateRect.Call(a.mainHWND, 0, 0)
	}
	a.mu.RLock()
	refUnix := a.lastCleanUnix
	if refUnix == 0 && !(cfg.AutoCleanThresholdEnabled && pct >= cfg.AutoCleanThresholdPct) {
		refUnix = a.startedUnix
	}
	a.mu.RUnlock()
	if ok, src := domain.ShouldAutoClean(cfg, pct, refUnix, time.Now().Unix()); ok {
		a.triggerClean(src, 0)
	}
}

func (a *AppController) runAsyncClean(execFn func() (domain.CleanResult, error)) {
	if a.engine.IsCleaning() {
		return
	}
	go func() {
		res, err := execFn()
		if err == nil {
			a.mu.Lock()
			a.pendingResult = &res
			a.lastCleanUnix = time.Now().Unix()
			a.mu.Unlock()
		}
		procPostMessageW.Call(a.mainHWND, wmAppCleanDone, 0, 0)
	}()
}

func (a *AppController) triggerClean(src domain.CleanSource, overrideMask domain.CleanRegion) {
	cfg := a.getConfig()
	if src == domain.SourceManual && cfg.ConfirmClean {
		if !ConfirmBox(a.mainHWND, "Confirmar Limpeza", "Deseja limpar RAM, Paginação, 2º plano e Kernel agora?") {
			return
		}
	}
	a.runAsyncClean(func() (domain.CleanResult, error) {
		return a.engine.Clean(context.Background(), src, cfg, overrideMask)
	})
}

func (a *AppController) onCleanFinished() {
	a.mu.Lock()
	res := a.pendingResult
	a.pendingResult = nil
	a.mu.Unlock()

	if res != nil {
		a.lastSummary = domain.FormatCleanSummary(*res)
		cfg := a.getConfig()
		if cfg.ShowBalloonResult {
			a.tray.ShowBalloon("RamKiller", a.lastSummary, cfg.NotificationSound)
		}
	}
	a.refreshStats()
	a.forceTrayRefresh()
	procInvalidateRect.Call(a.mainHWND, 0, 0)
}

func (a *AppController) showMainWindow() {
	procShowWindow.Call(a.mainHWND, SW_RESTORE)
	procSetForegroundWnd.Call(a.mainHWND)
	procInvalidateRect.Call(a.mainHWND, 0, 0)
}

func (a *AppController) toggleMainWindow() {
	if vis, _, _ := procIsWindowVisible.Call(a.mainHWND); vis != 0 {
		procShowWindow.Call(a.mainHWND, SW_HIDE)
		sys.TrimSelf()
		return
	}
	a.showMainWindow()
}

func (a *AppController) shutdownApp(pauseUntilReboot bool) {
	procKillTimer.Call(a.mainHWND, 1)
	procUnregisterHotKey.Call(a.mainHWND, 1)
	if pauseUntilReboot {
		_ = sys.PauseUntilReboot()
	} else {
		sys.SignalGuardianStop()
	}
	if a.tray != nil {
		a.tray.Destroy()
	}
	if a.settingsHWND != 0 {
		procDestroyWindow.Call(a.settingsHWND)
		a.settingsHWND = 0
	}
	procDestroyWindow.Call(a.mainHWND)
}
