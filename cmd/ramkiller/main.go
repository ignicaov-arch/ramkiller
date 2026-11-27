//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ignicaov-arch/ramkiller/internal/sys"
	"github.com/ignicaov-arch/ramkiller/internal/ui"
)

func hasArg(args []string, flag string) bool {
	for _, a := range args {
		if strings.EqualFold(strings.TrimSpace(a), flag) {
			return true
		}
	}
	return false
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		if code, handled := handleCLI(args); handled {
			os.Exit(code)
		}
	}

	if hasArg(args, "--guardian") {
		if err := sys.RunGuardian(); err != nil {
			os.Exit(1)
		}
		return
	}

	isBoot := hasArg(args, "--boot")
	minimized := hasArg(args, "--minimized")
	if isBoot {
		if sys.IsPausedForCurrentBoot() || sys.IsMutexHeld(sys.MainMutexName) {
			return
		}
		if !sys.IsElevated() && !hasArg(args, "--no-elevate") {
			if sys.IsSkipUACTaskEnabled() && sys.RunSkipUACTask() == nil {
				return
			}
			if err := sys.RelaunchAsAdmin(args...); err == nil {
				return
			}
		}
	} else {
		_ = sys.ClearPauseState()
		if !minimized && ui.FocusRunningInstance() {
			return
		}
		if !sys.IsElevated() && !hasArg(args, "--no-elevate") {
			if err := sys.RunInteractiveTask(); err == nil {
				return
			}
			if err := sys.RelaunchAsAdmin(args...); err == nil {
				return
			}
		}
	}

	if err := ui.RunApp(minimized, isBoot); err != nil {
		attachParentConsole()
		fmt.Fprintf(os.Stderr, "Erro ao iniciar RamKiller: %v\n", err)
		os.Exit(1)
	}
}
