package domain

import "strings"

// LaunchPlan define exatamente como uma invocacao do RamKiller deve se comportar sem abrir janelas indevidas.
type LaunchPlan struct {
	Minimized     bool
	IsBoot        bool
	NoElevate     bool
	Silent        bool
	RespectPause  bool
	FocusExisting bool
	UseSilentTask bool
	RelaunchArgs  []string
}

func hasFlagArg(args []string, flag string) bool {
	for _, a := range args {
		if strings.EqualFold(strings.TrimSpace(a), flag) {
			return true
		}
	}
	return false
}

// ResolveLaunchPlan traduz flags CLI em plano deterministico de inicializacao silenciosa vs interativa.
func ResolveLaunchPlan(args []string) LaunchPlan {
	minimized := hasFlagArg(args, "--minimized")
	isBoot := hasFlagArg(args, "--boot")
	noElevate := hasFlagArg(args, "--no-elevate")
	silent := minimized || isBoot

	var relaunch []string
	if minimized {
		relaunch = append(relaunch, "--minimized")
	}
	if isBoot {
		relaunch = append(relaunch, "--boot")
	}

	return LaunchPlan{
		Minimized:     minimized,
		IsBoot:        isBoot,
		NoElevate:     noElevate,
		Silent:        silent,
		RespectPause:  silent,
		FocusExisting: !silent,
		UseSilentTask: silent,
		RelaunchArgs:  relaunch,
	}
}

// ShouldFocusOnMutexCollision decide se uma segunda instancia colidindo no mutex pode exibir a janela ativa.
func ShouldFocusOnMutexCollision(startMinimized bool, isBoot bool) bool {
	return !startMinimized && !isBoot
}

// ShouldShowCleanBalloon evita notificacoes intrusivas quando limpezas automaticas ocorrem em tela cheia/jogo.
func ShouldShowCleanBalloon(cfg Config, src CleanSource, userAcceptsNotifications bool) bool {
	if !cfg.ShowBalloonResult {
		return false
	}
	if src == SourceAuto && !userAcceptsNotifications {
		return false
	}
	return true
}
