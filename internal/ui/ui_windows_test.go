//go:build windows

package ui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

func TestTrayIconCreationAndCleanup(t *testing.T) {
	cfg := domain.DefaultConfig()
	for _, pct := range []int{42, 68, 94} {
		img := RenderTrayIconNRGBA(pct, cfg, 32)
		if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
			t.Fatalf("dimensao invalida para %d%%: %v", pct, img.Bounds())
		}
		hIcon, err := CreateTrayHICON(pct, cfg, 32)
		if err != nil || hIcon == 0 {
			t.Fatalf("CreateTrayHICON(%d) falhou: %v", pct, err)
		}
		procDestroyIcon.Call(hIcon)
	}
}

func TestAllSettingsControlsAndSteppersWork(t *testing.T) {
	ctrl := &AppController{engine: sys.NewEngine()}
	cfg := domain.DefaultConfig()

	boolToggles := []struct {
		id  string
		get func(*domain.Config) bool
	}{
		{"chk_always_top", func(c *domain.Config) bool { return c.AlwaysOnTop }},
		{"chk_autorun", func(c *domain.Config) bool { return c.LoadOnStartup }},
		{"chk_start_menu", func(c *domain.Config) bool { return c.StartMenuShortcut }},
		{"chk_self_heal", func(c *domain.Config) bool { return c.SelfHeal }},
		{"chk_minimized", func(c *domain.Config) bool { return c.StartMinimized }},
		{"chk_confirm", func(c *domain.Config) bool { return c.ConfirmClean }},
		{"chk_kill_bg", func(c *domain.Config) bool { return c.KillBackgroundApps }},
		{"chk_auto_pct", func(c *domain.Config) bool { return c.AutoCleanThresholdEnabled }},
		{"chk_auto_min", func(c *domain.Config) bool { return c.AutoCleanIntervalEnabled }},
		{"chk_hotkey", func(c *domain.Config) bool { return c.HotkeyEnabled }},
		{"chk_tray_transp", func(c *domain.Config) bool { return c.TrayTransparent }},
		{"chk_tray_border", func(c *domain.Config) bool { return c.TrayShowBorder }},
		{"chk_tray_round", func(c *domain.Config) bool { return c.TrayRoundCorners }},
		{"chk_tray_chgbg", func(c *domain.Config) bool { return c.TrayChangeBg }},
		{"chk_tray_aa", func(c *domain.Config) bool { return c.TrayAntialias }},
		{"chk_balloon", func(c *domain.Config) bool { return c.ShowBalloonResult }},
		{"chk_sound", func(c *domain.Config) bool { return c.NotificationSound }},
		{"chk_log", func(c *domain.Config) bool { return c.LogResults }},
	}

	for _, tc := range boolToggles {
		before := tc.get(&cfg)
		ctrl.applySettingsMutation(0, tc.id, &cfg)
		if tc.get(&cfg) == before {
			t.Errorf("checkbox %s nao alternou valor", tc.id)
		}
	}

	ctrl.applySettingsMutation(0, "auto_pct_dec", &cfg)
	ctrl.applySettingsMutation(0, "auto_pct_inc", &cfg)
	ctrl.applySettingsMutation(0, "auto_min_dec", &cfg)
	if cfg.AutoCleanIntervalMin != 10 {
		t.Errorf("auto_min_dec esperado 10, obteve %d", cfg.AutoCleanIntervalMin)
	}
	cfg.AutoCleanIntervalMin = 5
	ctrl.applySettingsMutation(0, "auto_min_dec", &cfg)
	if cfg.AutoCleanIntervalMin != 1 {
		t.Errorf("step de 5 para 1 falhou: %d", cfg.AutoCleanIntervalMin)
	}
	ctrl.applySettingsMutation(0, "auto_min_inc", &cfg)
	if cfg.AutoCleanIntervalMin != 5 {
		t.Errorf("step de 1 para 5 falhou: %d", cfg.AutoCleanIntervalMin)
	}

	ctrl.applySettingsMutation(0, "anom_mb_inc", &cfg)
	if cfg.AnomalyLimitMB != 2816 {
		t.Errorf("anom_mb_inc esperado 2816, obteve %d", cfg.AnomalyLimitMB)
	}
	ctrl.applySettingsMutation(0, "font_sz_inc", &cfg)
	if cfg.TrayFontSize != 11 {
		t.Errorf("font_sz_inc esperado 11, obteve %d", cfg.TrayFontSize)
	}
	ctrl.applySettingsMutation(0, "cycle_hotkey", &cfg)
	if cfg.HotkeyModifiers != 0x0002 {
		t.Errorf("cycle_hotkey deveria avancar preset, obteve mods=%d", cfg.HotkeyModifiers)
	}
	ctrl.applySettingsMutation(0, "tray_single_act", &cfg)
	ctrl.applySettingsMutation(0, "tray_middle_act", &cfg)
	if cfg.TraySingleClick != domain.ActionCleanMemory || cfg.TrayMiddleClick != domain.ActionOpenTaskMgr {
		t.Errorf("acoes de clique nao avancaram: single=%d mid=%d", cfg.TraySingleClick, cfg.TrayMiddleClick)
	}

	cfg.WarningLevelPct = 85
	cfg.DangerLevelPct = 90
	ctrl.applySettingsMutation(0, "dang_lvl_dec", &cfg)
	cfg.Sanitize()
	if cfg.DangerLevelPct != 85 || cfg.WarningLevelPct != 80 {
		t.Errorf("dang_lvl_dec acoplado falhou: warn=%d dang=%d", cfg.WarningLevelPct, cfg.DangerLevelPct)
	}
}

func TestRenderQAScreenshotsAndAllButtonHits(t *testing.T) {
	outDir := t.TempDir()
	if err := RenderQAScreenshots(outDir); err != nil {
		t.Fatalf("RenderQAScreenshots falhou: %v", err)
	}
	expected := map[string][2]int{
		"main_window.png":         {mainWinWidth, mainWinHeight},
		"settings_general.png":    {settingsWinWidth, settingsWinHeight},
		"settings_memory.png":     {settingsWinWidth, settingsWinHeight},
		"settings_appearance.png": {settingsWinWidth, settingsWinHeight},
		"settings_tray.png":       {settingsWinWidth, settingsWinHeight},
		"tray_icons_strip.png":    {240, 88},
	}
	for name, dims := range expected {
		f, err := os.Open(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("arquivo esperado ausente %s: %v", name, err)
		}
		cfgImg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Fatalf("PNG invalido %s: %v", name, err)
		}
		if cfgImg.Width != dims[0] || cfgImg.Height != dims[1] {
			t.Errorf("%s dimensoes %dx%d != esperado %dx%d", name, cfgImg.Width, cfgImg.Height, dims[0], dims[1])
		}
	}

	// Confirma botoes obrigatorios e acoes de processo no Canvas Principal
	cfg := domain.DefaultConfig()
	stats := domain.MemoryStats{
		Elevated: true,
		TopProcesses: []domain.ProcessInfo{
			{PID: 4242, Name: "chrome.exe", WorkingSetBytes: 850 * 1024 * 1024, PrivateBytes: 1200 * 1024 * 1024, OpenWindow: true},
			{PID: 5353, Name: "widgets.exe", WorkingSetBytes: 320 * 1024 * 1024, PrivateBytes: 600 * 1024 * 1024, Killable: true},
		},
	}
	mc, hits := RenderMainCanvas(stats, cfg, "", "", false)
	mc.Close()
	needed := map[string]bool{"open_settings": false, "clean_now": false, "clean_pagefile": false, "stop_until_reboot": false, "trim_pid_4242": false, "kill_pid_5353": false}
	for _, h := range hits {
		if h.ID == "clean_deep" {
			t.Errorf("clean_deep deveria ter sido unificado em clean_now")
		}
		if _, ok := needed[h.ID]; ok {
			needed[h.ID] = true
		}
	}
	for id, found := range needed {
		if !found {
			t.Errorf("botao %s ausente nos hits do painel principal", id)
		}
	}
}
