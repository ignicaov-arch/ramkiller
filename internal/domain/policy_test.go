package domain

import (
	"strings"
	"testing"
	"unicode/utf16"
)

func TestEffectiveMaskUnifiedAcrossSources(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CleanMask = MaskAll
	cfg.AllowStandbyInAuto = false

	for _, src := range []CleanSource{SourceManual, SourceAuto, SourceHotkey, SourceCLI} {
		if got := EffectiveMask(cfg, src, 0); got != MaskAll {
			t.Fatalf("fonte %s deve aplicar MaskAll unificado, obteve %v", src, got)
		}
	}
	if got := EffectiveMask(cfg, SourceAuto, RegionWorkingSet); got != RegionWorkingSet {
		t.Fatalf("override explicitamente informado deve prevalecer, obteve %v", got)
	}
	cfg.CleanMask = 0
	if fallback := EffectiveMask(cfg, SourceAuto, 0); fallback != MaskAll {
		t.Fatalf("esperava fallback MaskAll, obteve %v", fallback)
	}
}

func TestShouldAutoCleanCooldownAndTriggers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AutoCleanThresholdEnabled = true
	cfg.AutoCleanThresholdPct = 80
	cfg.AutoCleanIntervalEnabled = true
	cfg.AutoCleanIntervalMin = 10

	if ok, _ := ShouldAutoClean(cfg, 75, 1000, 1200); ok {
		t.Error("nao devia disparar abaixo do threshold e antes do intervalo")
	}
	if ok, src := ShouldAutoClean(cfg, 82, 0, 1000); !ok || src != SourceAuto {
		t.Errorf("devia disparar no primeiro threshold: ok=%v src=%v", ok, src)
	}
	if ok, _ := ShouldAutoClean(cfg, 85, 1000, 1030); ok {
		t.Error("cooldown de 60s violado")
	}
	if ok, _ := ShouldAutoClean(cfg, 85, 1000, 1060); !ok {
		t.Error("deveria liberar exatamente em 60s")
	}
	if ok, _ := ShouldAutoClean(cfg, 50, 1000, 1600); !ok {
		t.Error("deveria disparar por intervalo de 10 minutos (600s)")
	}
}

func TestProtectedProcessAndAnomaly(t *testing.T) {
	if !IsProtectedProcess("chrome.exe", 4, 999, 888, true) {
		t.Error("PID <= 4 deve ser protegido")
	}
	if !IsProtectedProcess("node.exe", 999, 999, 888, true) {
		t.Error("selfPID deve ser protegido")
	}
	if !IsProtectedProcess("League of Legends.exe", 555, 999, 555, true) {
		t.Error("janela em foco deve ser protegida quando protectForeground=true")
	}
	if IsProtectedProcess("League of Legends.exe", 555, 999, 555, false) {
		t.Error("nao devia proteger foreground quando protectForeground=false")
	}
	if IsProtectedProcess("chrome.exe", 1200, 999, 888, true) {
		t.Error("chrome sem janela aberta nao e processo de nucleo")
	}
	if !IsProtectedProcess("AnyDesk.exe", 4500, 999, 888, true) {
		t.Error("AnyDesk.exe deve ser protegido contra encerramento em segundo plano")
	}
	if !IsProtectedProcess("TeamViewer.exe", 4501, 999, 888, false) {
		t.Error("TeamViewer.exe deve ser protegido contra encerramento em segundo plano")
	}
	if !IsAnomalyProcess(3*gb, 2560) {
		t.Error("3 GB deve sinalizar anomalia para teto 2560 MB")
	}
	if IsAnomalyProcess(500*mb, 2560) {
		t.Error("500 MB nao deve sinalizar anomalia")
	}
}

func TestFormattingAndTooltipUTF16Cap(t *testing.T) {
	cfg := DefaultConfig()
	if SeverityForPercent(45, cfg) != SeverityNormal || SeverityForPercent(60, cfg) != SeverityWarning || SeverityForPercent(90, cfg) != SeverityDanger {
		t.Error("niveis de severidade incorretos")
	}

	stats := MemoryStats{
		Physical:    BankStat{UsedBytes: 8 * gb, FreeBytes: 8 * gb, TotalBytes: 16 * gb, Percent: 50},
		Pagefile:    BankStat{Percent: 45},
		SystemCache: BankStat{Percent: 7},
	}
	tip := FormatTooltip(stats)
	if len(utf16.Encode([]rune(tip))) > 120 {
		t.Fatalf("tooltip excedeu 120 UTF-16 units")
	}
	if !strings.Contains(tip, "50%") || !strings.Contains(tip, "Paginação: 45%") {
		t.Errorf("tooltip inesperado: %q", tip)
	}
	if got := TruncateUTF16(strings.Repeat("😀", 80), 120); len(utf16.Encode([]rune(got))) > 120 {
		t.Errorf("truncamento UTF-16 falhou com surrogates")
	}
	if lbl := FormatHotkeyLabel(0x0002|0x0004, 0x70); lbl != "Ctrl + Shift + F1" {
		t.Errorf("hotkey inesperado: %q", lbl)
	}
	if lbl := FormatHotkeyLabel(0x0001, 'A'); lbl != "Alt + A" {
		t.Errorf("hotkey Alt+A inesperado: %q", lbl)
	}
	if FormatBytes(512) != "512 B" || FormatBytes(2048) != "2 KB" {
		t.Errorf("FormatBytes B/KB inesperado")
	}
	if len(AllRegionSpecs()) != 8 || len(RegionLabels(MaskAll)) != 8 {
		t.Errorf("esperava 8 regioes configuradas")
	}
	sum := FormatCleanSummary(CleanResult{FreedBytes: 2 * gb, PagefileFreedBytes: 512 * mb, KilledProcs: 7, TrimmedProcs: 140, KernelExecuted: true, DurationMs: 35})
	if !strings.Contains(sum, "2.00 GB RAM") || !strings.Contains(sum, "512 MB Paginação") || !strings.Contains(sum, "7 encerrados") || !strings.Contains(sum, "Modo Kernel") {
		t.Errorf("resumo unificado inesperado: %q", sum)
	}
}
