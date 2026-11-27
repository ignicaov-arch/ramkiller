//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
	"github.com/ignicaov-arch/ramkiller/internal/sys"
	"github.com/ignicaov-arch/ramkiller/internal/ui"
)

var procAttachConsole = sys.Kernel32.NewProc("AttachConsole")

func attachParentConsole() {
	const attachParentProcess = ^uintptr(0)
	procAttachConsole.Call(attachParentProcess)
}

func handleCLI(args []string) (int, bool) {
	var wantStatus, wantClean, wantFull, wantPagefile, wantAudit, wantShortcut, wantPause, wantJSON, wantHelp bool
	var renderQADir string

	for _, raw := range args {
		a := strings.TrimSpace(raw)
		switch {
		case a == "--status":
			wantStatus = true
		case a == "--clean":
			wantClean = true
		case a == "--clean-full":
			wantFull = true
		case a == "--clean-pagefile":
			wantPagefile = true
		case a == "--audit-procs":
			wantAudit = true
		case a == "--install-shortcut":
			wantShortcut = true
		case a == "--pause-until-reboot":
			wantPause = true
		case a == "--json":
			wantJSON = true
		case a == "--help" || a == "-h":
			wantHelp = true
		case strings.HasPrefix(a, "--render-qa="):
			renderQADir = strings.TrimPrefix(a, "--render-qa=")
		}
	}

	if !wantStatus && !wantClean && !wantFull && !wantPagefile && !wantAudit && !wantShortcut && !wantPause && !wantHelp && renderQADir == "" {
		return 0, false
	}

	attachParentConsole()
	if wantHelp {
		printHelp()
		return 0, true
	}
	if wantPause {
		return runPauseCommand(wantJSON), true
	}
	if renderQADir != "" {
		if err := ui.RenderQAScreenshots(renderQADir); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao gerar QA PNGs: %v\n", err)
			return 1, true
		}
		fmt.Printf("QA PNGs salvos em %s\n", renderQADir)
		return 0, true
	}

	cfg, _ := sys.LoadConfig()
	if wantAudit {
		return runAuditCommand(cfg, wantJSON), true
	}
	if wantShortcut {
		return runInstallShortcutCommand(cfg, wantJSON), true
	}
	if wantPagefile {
		eng := sys.NewEngine()
		res, err := eng.CleanPagefile(context.Background(), cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao limpar paginação: %v\n", err)
			return 1, true
		}
		printOutput(res, wantJSON, func() {
			fmt.Printf("RamKiller CLI: %s\n", domain.FormatCleanSummary(res))
			fmt.Printf("Pagefile usado: %s -> %s\n", domain.FormatBytes(res.BeforePagefileBytes), domain.FormatBytes(res.AfterPagefileBytes))
		})
		return 0, true
	}
	if wantClean || wantFull {
		mask := cfg.CleanMask
		if wantFull {
			mask = domain.MaskAll
		}
		eng := sys.NewEngine()
		res, err := eng.Clean(context.Background(), domain.SourceCLI, cfg, mask)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro na limpeza: %v\n", err)
			return 1, true
		}
		printOutput(res, wantJSON, func() {
			fmt.Printf("RamKiller CLI: %s\n", domain.FormatCleanSummary(res))
			fmt.Printf("RAM usada: %s -> %s | Pagefile usado: %s -> %s\n",
				domain.FormatBytes(res.BeforeUsedBytes), domain.FormatBytes(res.AfterUsedBytes),
				domain.FormatBytes(res.BeforePagefileBytes), domain.FormatBytes(res.AfterPagefileBytes))
		})
		return 0, true
	}

	stats, err := sys.ReadMemoryStats(5, cfg.AnomalyLimitMB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler telemetria: %v\n", err)
		return 1, true
	}
	printOutput(stats, wantJSON, func() {
		fmt.Printf("RamKiller Status | Fisica: %d%% (%s/%s) | Paginacao: %d%% | Cache: %d%%\n",
			stats.Physical.Percent, domain.FormatBytes(stats.Physical.UsedBytes),
			domain.FormatBytes(stats.Physical.TotalBytes), stats.Pagefile.Percent, stats.SystemCache.Percent)
		for _, p := range stats.TopProcesses {
			fmt.Printf("  - %-26s PID %-6d Privado: %-9s WS: %s\n",
				p.Name, p.PID, domain.FormatBytes(p.PrivateBytes), domain.FormatBytes(p.WorkingSetBytes))
		}
	})
	return 0, true
}

func runPauseCommand(asJSON bool) int {
	err := sys.PauseUntilReboot()
	out := struct {
		Paused bool   `json:"pausedUntilReboot"`
		BootID int64  `json:"bootId"`
		Error  string `json:"error,omitempty"`
	}{Paused: err == nil, BootID: sys.GetBootSessionID()}
	if err != nil {
		out.Error = err.Error()
		printOutput(out, asJSON, func() { fmt.Fprintf(os.Stderr, "Erro ao pausar: %v\n", err) })
		return 1
	}
	printOutput(out, asJSON, func() { fmt.Println("RamKiller encerrado e pausado até reiniciar o PC.") })
	return 0
}

func printOutput(v any, asJSON bool, textFn func()) {
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(v)
		return
	}
	textFn()
}

func printHelp() {
	fmt.Println("RamKiller - Limpador nativo de RAM e Paginação para Windows")
	fmt.Println("Uso: --minimized | --status | --audit-procs | --clean | --clean-pagefile | --install-shortcut | --pause-until-reboot | --guardian | --json")
}
