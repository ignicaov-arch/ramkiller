package domain

import (
	"reflect"
	"testing"
)

func TestResolveLaunchPlanInteractiveVsSilent(t *testing.T) {
	interactive := ResolveLaunchPlan(nil)
	if interactive.Silent || interactive.RespectPause || !interactive.FocusExisting || interactive.UseSilentTask {
		t.Fatalf("plano interativo inesperado: %+v", interactive)
	}
	if len(interactive.RelaunchArgs) != 0 {
		t.Fatalf("plano interativo nao deveria repassar args extras: %v", interactive.RelaunchArgs)
	}

	// Caso exato das 20:20:34: disparo --minimized sem --boot NUNCA pode usar RamKiller_Interactive nem focar.
	minOnly := ResolveLaunchPlan([]string{"--minimized"})
	if !minOnly.Minimized || minOnly.IsBoot || !minOnly.Silent || !minOnly.RespectPause || minOnly.FocusExisting || !minOnly.UseSilentTask {
		t.Fatalf("disparo --minimized deveria permanecer 100%% silencioso: %+v", minOnly)
	}
	if !reflect.DeepEqual(minOnly.RelaunchArgs, []string{"--minimized"}) {
		t.Fatalf("RelaunchArgs inesperado para --minimized: %v", minOnly.RelaunchArgs)
	}

	bootFull := ResolveLaunchPlan([]string{" --MINIMIZED ", "--boot", "--no-elevate"})
	if !bootFull.Minimized || !bootFull.IsBoot || !bootFull.NoElevate || !bootFull.Silent || !bootFull.UseSilentTask || bootFull.FocusExisting {
		t.Fatalf("disparo --minimized --boot inesperado: %+v", bootFull)
	}
	if !reflect.DeepEqual(bootFull.RelaunchArgs, []string{"--minimized", "--boot"}) {
		t.Fatalf("RelaunchArgs inesperado para --minimized --boot: %v", bootFull.RelaunchArgs)
	}

	bootOnly := ResolveLaunchPlan([]string{"--boot"})
	if bootOnly.Minimized || !bootOnly.IsBoot || !bootOnly.Silent || !bootOnly.RespectPause || bootOnly.FocusExisting || !bootOnly.UseSilentTask {
		t.Fatalf("disparo --boot inesperado: %+v", bootOnly)
	}
}

func TestShouldFocusOnMutexCollision(t *testing.T) {
	if !ShouldFocusOnMutexCollision(false, false) {
		t.Error("abertura manual sem flags deveria focar instancia existente")
	}
	if ShouldFocusOnMutexCollision(true, false) {
		t.Error("--minimized nao pode jamais enviar wmAppShow na colisao de mutex")
	}
	if ShouldFocusOnMutexCollision(false, true) {
		t.Error("--boot nao pode enviar wmAppShow na colisao de mutex")
	}
	if ShouldFocusOnMutexCollision(true, true) {
		t.Error("--minimized --boot nao pode enviar wmAppShow na colisao de mutex")
	}
}

func TestShouldShowCleanBalloon(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ShowBalloonResult = false
	if ShouldShowCleanBalloon(cfg, SourceManual, true) {
		t.Error("balao desativado na config deve bloquear notificacoes")
	}

	cfg.ShowBalloonResult = true
	if ShouldShowCleanBalloon(cfg, SourceAuto, false) {
		t.Error("limpeza automatica durante tela cheia/jogo/apresentacao nao pode exibir balao")
	}
	if !ShouldShowCleanBalloon(cfg, SourceAuto, true) {
		t.Error("limpeza automatica em area de trabalho normal deve respeitar ShowBalloonResult")
	}
	if !ShouldShowCleanBalloon(cfg, SourceManual, false) || !ShouldShowCleanBalloon(cfg, SourceHotkey, false) {
		t.Error("acao explicita do usuario pode retornar feedback visual")
	}
}
