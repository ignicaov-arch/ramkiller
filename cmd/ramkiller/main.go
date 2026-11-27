//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
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

	plan := domain.ResolveLaunchPlan(args)
	if plan.RespectPause {
		if sys.IsPausedForCurrentBoot() {
			return
		}
	} else {
		_ = sys.ClearPauseState()
	}

	if sys.IsMutexHeld(sys.MainMutexName) {
		if plan.FocusExisting {
			_ = ui.FocusRunningInstance()
		}
		return
	}

	if !sys.IsElevated() && !plan.NoElevate {
		if plan.UseSilentTask {
			if sys.IsSkipUACTaskEnabled() && sys.RunSkipUACTask() == nil {
				return
			}
		} else {
			if err := sys.RunInteractiveTask(); err == nil {
				return
			}
		}
		if err := sys.RelaunchAsAdmin(plan.RelaunchArgs...); err == nil {
			return
		}
	}

	if err := ui.RunApp(plan.Minimized, plan.IsBoot); err != nil {
		attachParentConsole()
		fmt.Fprintf(os.Stderr, "Erro ao iniciar RamKiller: %v\n", err)
		os.Exit(1)
	}
}
