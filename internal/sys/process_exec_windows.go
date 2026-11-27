//go:build windows

package sys

import (
	"fmt"
	"os"
	"sort"
	"syscall"
	"time"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const quotaHardWSMinDisable = 0x00000002

func lookupProcessName(pid uint32) string {
	entries, err := enumSnapshotProcesses()
	if err != nil {
		return fmt.Sprintf("PID %d", pid)
	}
	for i := range entries {
		if entries[i].ProcessID == pid {
			return syscall.UTF16ToString(entries[i].ExeFile[:])
		}
	}
	return fmt.Sprintf("PID %d", pid)
}

// TrimProcessWorkingSetDelta recicla paginas fisicas do processo e retorna o delta real em Working Set.
func TrimProcessWorkingSetDelta(pid uint32) (name string, beforeWS, afterWS uint64, err error) {
	if pid <= 4 {
		return "", 0, 0, fmt.Errorf("PID protegido: %d", pid)
	}
	name = lookupProcessName(pid)
	beforeWS, _, _, _ = readProcDetails(pid, false)

	h, _, openErr := procOpenProcess.Call(procQueryLimitedInfo|procSetQuota, 0, uintptr(pid))
	if h == 0 {
		return name, beforeWS, beforeWS, fmt.Errorf("OpenProcess(%d): %w", pid, openErr)
	}
	defer procCloseHandle.Call(h)

	maxNeg := ^uintptr(0)
	procSetProcessWorkingSetEx.Call(h, maxNeg, maxNeg, quotaHardWSMinDisable)
	r, _, callErr := procK32EmptyWorkingSet.Call(h)
	if r == 0 {
		r, _, callErr = procPsapiEmptyWorkingSet.Call(h)
	}
	procSetProcessWorkingSetEx.Call(h, maxNeg, maxNeg, 0)
	if r == 0 {
		return name, beforeWS, beforeWS, fmt.Errorf("EmptyWorkingSet(%d): %w", pid, callErr)
	}
	time.Sleep(15 * time.Millisecond)
	afterWS, _, _, okAfter := readProcDetails(pid, false)
	if !okAfter {
		afterWS = beforeWS
	}
	return name, beforeWS, afterWS, nil
}

// TrimProcessWorkingSet recicla paginas inativas em lote em alta velocidade (sem sleep nem snapshot O(N^2)).
func TrimProcessWorkingSet(pid uint32) error {
	if pid <= 4 {
		return fmt.Errorf("PID protegido: %d", pid)
	}
	h, _, openErr := procOpenProcess.Call(procQueryLimitedInfo|procSetQuota, 0, uintptr(pid))
	if h == 0 {
		return fmt.Errorf("OpenProcess(%d): %w", pid, openErr)
	}
	defer procCloseHandle.Call(h)

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
		if s.IsNetService {
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
		recycled, recNames := RecycleLeakyPagefileHosts(snaps, GetForegroundPID())
		killed += recycled
		for _, n := range recNames {
			if _, ok := seenNames[n]; !ok && len(killedNames) < 20 {
				seenNames[n] = struct{}{}
				killedNames = append(killedNames, n)
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

// ListTopProcesses retorna os maiores consumidores reais de RAM fisica (Working Set) com classificacao cirurgica.
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
			Anomaly:         domain.IsAnomalyProcess(s.WorkingSetBytes, anomalyLimitMB) || domain.IsAnomalyProcess(s.PrivateBytes, anomalyLimitMB),
		})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].WorkingSetBytes != list[j].WorkingSetBytes {
			return list[i].WorkingSetBytes > list[j].WorkingSetBytes
		}
		return list[i].PrivateBytes > list[j].PrivateBytes
	})
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}
