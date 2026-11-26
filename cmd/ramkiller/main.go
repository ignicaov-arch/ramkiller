//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ignicaov-arch/ramkiller/internal/ui"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		if code, handled := handleCLI(args); handled {
			os.Exit(code)
		}
	}

	minimized := false
	for _, a := range args {
		if strings.EqualFold(strings.TrimSpace(a), "--minimized") {
			minimized = true
		}
	}

	if err := ui.RunApp(minimized); err != nil {
		attachParentConsole()
		fmt.Fprintf(os.Stderr, "Erro ao iniciar RamKiller: %v\n", err)
		os.Exit(1)
	}
}
