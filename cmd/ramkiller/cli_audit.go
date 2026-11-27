//go:build windows

package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

type auditEntry struct {
	PID        uint32            `json:"pid"`
	Name       string            `json:"name"`
	Action     domain.ProcAction `json:"action"`
	Reason     string            `json:"reason"`
	OpenWindow bool              `json:"openWindow"`
	NetActive  bool              `json:"netActive,omitempty"`
	PrivateMB  uint64            `json:"privateMb"`
}

func runAuditCommand(cfg domain.Config, asJSON bool) int {
	snaps, deskOK, err := sys.CollectProcessSnapshots()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro na auditoria de processos: %v\n", err)
		return 1
	}
	decs := domain.ClassifyProcesses(snaps, uint32(os.Getpid()), deskOK, cfg.AnomalyLimitMB)
	var rows []auditEntry
	for _, s := range snaps {
		if s.SessionID == 0 || s.PID <= 4 {
			continue
		}
		d := decs[s.PID]
		rows = append(rows, auditEntry{
			PID:        s.PID,
			Name:       s.Name,
			Action:     d.Action,
			Reason:     d.Reason,
			OpenWindow: s.HasOpenWindow,
			NetActive:  s.HasActiveNet || s.IsNetService,
			PrivateMB:  s.PrivateBytes / (1024 * 1024),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].PrivateMB > rows[j].PrivateMB })
	payload := struct {
		DesktopEnumOK bool         `json:"desktopEnumOk"`
		Processes     []auditEntry `json:"processes"`
	}{DesktopEnumOK: deskOK, Processes: rows}
	printOutput(payload, asJSON, func() {
		fmt.Printf("DesktopEnumOK=%t | Processos usuario=%d\n", deskOK, len(rows))
		for _, r := range rows {
			netFlag := "     "
			if r.NetActive {
				netFlag = "[NET]"
			}
			fmt.Printf("  %-26s PID %-6d %-11s %s %4d MB  [%s]\n", r.Name, r.PID, r.Action, netFlag, r.PrivateMB, r.Reason)
		}
	})
	return 0
}

func runInstallShortcutCommand(cfg domain.Config, asJSON bool) int {
	sys.SyncStartupRegistration(cfg)
	lnkPath, _ := sys.StartMenuShortcutPath()
	payload := struct {
		ShortcutPath    string `json:"shortcutPath"`
		ShortcutExists  bool   `json:"shortcutExists"`
		AutorunEnabled  bool   `json:"autorunEnabled"`
		SkipUACTask     bool   `json:"skipUacTask"`
	}{
		ShortcutPath:   lnkPath,
		ShortcutExists: sys.IsStartMenuShortcutPresent(),
		AutorunEnabled: sys.IsAutorunEnabled(),
		SkipUACTask:    sys.IsSkipUACTaskEnabled(),
	}
	printOutput(payload, asJSON, func() {
		fmt.Printf("Atalho Menu Iniciar (%t): %s\n", payload.ShortcutExists, payload.ShortcutPath)
		fmt.Printf("Iniciar com Windows: Run=%t | TarefaElevada=%t\n", payload.AutorunEnabled, payload.SkipUACTask)
	})
	return 0
}
