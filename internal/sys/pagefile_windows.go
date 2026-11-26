//go:build windows

package sys

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const minLeakyCommitBytes uint64 = 160 * 1024 * 1024

var leakyCommitHostNames = map[string]struct{}{
	"dtsapo4service.exe":          {},
	"textinputhost.exe":           {},
	"startmenuexperiencehost.exe": {},
	"shellexperiencehost.exe":     {},
	"searchapp.exe":               {},
	"phoneexperiencehost.exe":     {},
}

// RecycleLeakyPagefileHosts reinicia hosts auto-regeneraveis com vazamento alto no pagefile (>= 160 MB).
func RecycleLeakyPagefileHosts(snaps []domain.ProcSnapshot, fgPID uint32) (recycled int, names []string) {
	seen := make(map[string]struct{})
	selfPID := uint32(os.Getpid())
	for _, s := range snaps {
		if s.PID <= 4 || s.PID == selfPID || (fgPID != 0 && s.PID == fgPID) || s.HasOpenWindow {
			continue
		}
		norm := strings.ToLower(strings.TrimSpace(s.Name))
		if _, ok := leakyCommitHostNames[norm]; !ok {
			continue
		}
		if s.PrivateBytes < minLeakyCommitBytes {
			continue
		}
		if err := TerminateProcessSafe(s.PID); err == nil {
			recycled++
			if _, exists := seen[s.Name]; !exists && len(names) < 20 {
				seen[s.Name] = struct{}{}
				names = append(names, s.Name)
			}
		}
	}
	return recycled, names
}

// ExecutePagefileProcessPlan encerra processos ociosos de 2o plano e recicla hosts com commit inflado, sem empurrar RAM aberta para o pagefile.
func ExecutePagefileProcessPlan(cfg domain.Config) (killed int, killedNames []string) {
	snaps, deskOK, err := CollectProcessSnapshots()
	if err != nil {
		return 0, nil
	}
	seen := make(map[string]struct{})
	if cfg.KillBackgroundApps {
		decisions := domain.ClassifyProcesses(snaps, uint32(os.Getpid()), deskOK, cfg.AnomalyLimitMB)
		for _, s := range snaps {
			if decisions[s.PID].Action != domain.ProcKill {
				continue
			}
			if err := TerminateProcessSafe(s.PID); err == nil {
				killed++
				if _, ok := seen[s.Name]; !ok && len(killedNames) < 20 {
					seen[s.Name] = struct{}{}
					killedNames = append(killedNames, s.Name)
				}
			}
		}
	}
	recycled, recNames := RecycleLeakyPagefileHosts(snaps, GetForegroundPID())
	killed += recycled
	for _, n := range recNames {
		if _, ok := seen[n]; !ok && len(killedNames) < 20 {
			seen[n] = struct{}{}
			killedNames = append(killedNames, n)
		}
	}
	return killed, killedNames
}

// RunPagefileKernelClean consolida caches de disco, registro e paginas identicas sem gravar novas paginas no pagefile.sys.
func RunPagefileKernelClean() (executed bool, warnings []string) {
	if !IsElevated() {
		return false, []string{"Limpeza completa de paginação requer Modo Kernel (Administrador)."}
	}
	var successes int
	runCmd := func(label string, fn func() uint32) {
		if st := fn(); st == 0 {
			successes++
		} else {
			warnings = append(warnings, fmt.Sprintf("%s retornou NTSTATUS 0x%08X", label, st))
		}
	}
	if flushFixedVolumes() > 0 {
		successes++
	}
	runCmd("SystemFileCache", flushSystemFileCache)
	runCmd("RegistryCache", func() uint32 {
		st, _, _ := procNtSetSystemInformation.Call(sysRegistryReconcileClass, 0, 0)
		return uint32(st)
	})
	runCmd("CombineMemoryLists", func() uint32 {
		var info memoryCombineInfoEx
		st, _, _ := procNtSetSystemInformation.Call(
			sysCombinePhysMemInfoClass,
			uintptr(unsafe.Pointer(&info)),
			unsafe.Sizeof(info),
		)
		return uint32(st)
	})
	runCmd("StandbyPriority0", func() uint32 { return ntMemoryListCommand(memCmdPurgeLowPriorityStandby) })
	return successes > 0, warnings
}
