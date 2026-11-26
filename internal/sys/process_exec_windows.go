//go:build windows

package sys

import (
	"fmt"
	"os"
	"sort"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

// TrimProcessWorkingSet recicla paginas inativas de um processo com SetProcessWorkingSetSizeEx + EmptyWorkingSet.
func TrimProcessWorkingSet(pid uint32) error {
	if pid <= 4 {
		return fmt.Errorf("PID protegido: %d", pid)
	}
	h, _, err := procOpenProcess.Call(procQueryLimitedInfo|procSetQuota, 0, uintptr(pid))
	if h == 0 {
		return fmt.Errorf("OpenProcess(%d): %w", pid, err)
	}
	defer procCloseHandle.Call(h)

	maxNeg := ^uintptr(0)
	procSetProcessWorkingSetEx.Call(h, maxNeg, maxNeg, 0)
	r, _, callErr := procK32EmptyWorkingSet.Call(h)
	if r == 0 {
		r, _, callErr = procPsapiEmptyWorkingSet.Call(h)
	}
	if r == 0 {
		return fmt.Errorf("EmptyWorkingSet(%d): %w", pid, callErr)
	}
	return nil
}

// TerminateProcessSafe encerra um processo descartavel recusando PID <= 4 e o proprio processo.
func TerminateProcessSafe(pid uint32) error {
	if pid <= 4 || pid == uint32(os.Getpid()) {
		return fmt.Errorf("recusado encerrar PID protegido: %d", pid)
	}
	h, _, err := procOpenProcess.Call(procTerminate, 0, uintptr(pid))
	if h == 0 {
		return fmt.Errorf("OpenProcess(TERMINATE, %d): %w", pid, err)
	}
	defer procCloseHandle.Call(h)

	r, _, termErr := procTerminateProcess.Call(h, 1)
	if r == 0 {
		return fmt.Errorf("TerminateProcess(%d): %w", pid, termErr)
	}
	return nil
}

// ExecuteProcessPlan encerra apenas processos ociosos de segundo plano e recicla os processos preservados.
func ExecuteProcessPlan(cfg domain.Config, allowKill bool) (killed int, killedNames []string, trimmed int) {
	snaps, deskOK, err := CollectProcessSnapshots()
	if err != nil {
		return 0, nil, 0
	}
	selfPID := uint32(os.Getpid())
	decisions := domain.ClassifyProcesses(snaps, selfPID, deskOK, cfg.AnomalyLimitMB)

	var killQueue []domain.ProcSnapshot
	var trimQueue []domain.ProcSnapshot
	for _, s := range snaps {
		dec := decisions[s.PID]
		if allowKill && dec.Action == domain.ProcKill {
			killQueue = append(killQueue, s)
			continue
		}
		if dec.Action == domain.ProcKeepAndTrim || (s.PID > 4 && (s.Name == "explorer.exe" || s.Name == "language_server.exe")) {
			trimQueue = append(trimQueue, s)
		}
	}

	if allowKill {
		seenNames := make(map[string]struct{})
		for _, s := range killQueue {
			if err := TerminateProcessSafe(s.PID); err == nil {
				killed++
				if _, ok := seenNames[s.Name]; !ok && len(killedNames) < 20 {
					seenNames[s.Name] = struct{}{}
					killedNames = append(killedNames, s.Name)
				}
			}
		}
	}

	for _, s := range trimQueue {
		if err := TrimProcessWorkingSet(s.PID); err == nil {
			trimmed++
		}
	}
	return killed, killedNames, trimmed
}

// TrimSelf reduz o working set do proprio processo RamKiller.
func TrimSelf() {
	h, _, _ := procGetCurrentProcess.Call()
	if h != 0 {
		r, _, _ := procK32EmptyWorkingSet.Call(h)
		if r == 0 {
			procPsapiEmptyWorkingSet.Call(h)
		}
	}
}

// ListTopProcesses retorna os maiores consumidores de memoria privada com classificacao cirurgica.
func ListTopProcesses(limit int, selfPID uint32, anomalyLimitMB int) ([]domain.ProcessInfo, error) {
	snaps, deskOK, err := CollectProcessSnapshots()
	if err != nil {
		return nil, err
	}
	decisions := domain.ClassifyProcesses(snaps, selfPID, deskOK, anomalyLimitMB)
	list := make([]domain.ProcessInfo, 0, len(snaps))
	for _, s := range snaps {
		if s.PID <= 4 || (s.WorkingSetBytes == 0 && s.PrivateBytes == 0) {
			continue
		}
		dec := decisions[s.PID]
		list = append(list, domain.ProcessInfo{
			PID:             s.PID,
			Name:            s.Name,
			PrivateBytes:    s.PrivateBytes,
			WorkingSetBytes: s.WorkingSetBytes,
			Protected:       dec.Action == domain.ProcKeepSystem,
			OpenWindow:      dec.Action != domain.ProcKill && (s.HasOpenWindow || dec.Reason == "janela aberta ou árvore ativa" || dec.Action == domain.ProcKeepAndTrim),
			Killable:        dec.Action == domain.ProcKill,
			Anomaly:         domain.IsAnomalyProcess(s.PrivateBytes, anomalyLimitMB),
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].PrivateBytes > list[j].PrivateBytes })
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}
