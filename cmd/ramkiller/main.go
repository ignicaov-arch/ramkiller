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

	if hasArg(args, "--boot") {
		if sys.IsPausedForCurrentBoot() {
			return
		}
	} else {
		_ = sys.ClearPauseState()
	}

	if !sys.IsElevated() && !hasArg(args, "--no-elevate") {
		if err := sys.RelaunchAsAdmin(args...); err == nil {
			return
		}
	}

	minimized := hasArg(args, "--minimized")
	if err := ui.RunApp(minimized); err != nil {
		attachParentConsole()
		fmt.Fprintf(os.Stderr, "Erro ao iniciar RamKiller: %v\n", err)
		os.Exit(1)
	}
}
