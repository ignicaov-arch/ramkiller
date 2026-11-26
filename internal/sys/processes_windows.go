//go:build windows

package sys

import (
	"fmt"
	"sort"
	"syscall"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const (
	th32csSnapProcess      = 0x00000002
	procQueryLimitedInfo   = 0x1000
	procVMRead             = 0x0010
	procSetQuota           = 0x0100
	invalidHandleValue     = ^uintptr(0)
)

type processEntry32W struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

type processMemCountersEx struct {
	Cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

// GetForegroundPID retorna o PID da janela atualmente em foco no Windows.
func GetForegroundPID() uint32 {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return 0
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}

func enumSnapshotProcesses() ([]processEntry32W, error) {
	snap, _, err := procCreateToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if snap == 0 || snap == invalidHandleValue {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot falhou: %w", err)
	}
	defer procCloseHandle.Call(snap)

	var entries []processEntry32W
	var pe processEntry32W
	pe.Size = uint32(unsafe.Sizeof(pe))
	r, _, _ := procProcess32FirstW.Call(snap, uintptr(unsafe.Pointer(&pe)))
	for r != 0 {
		entries = append(entries, pe)
		pe.Size = uint32(unsafe.Sizeof(pe))
		r, _, _ = procProcess32NextW.Call(snap, uintptr(unsafe.Pointer(&pe)))
	}
	return entries, nil
}

func readProcMemory(pid uint32) (ws uint64, priv uint64, ok bool) {
	h, _, _ := procOpenProcess.Call(procQueryLimitedInfo|procVMRead, 0, uintptr(pid))
	if h == 0 {
		return 0, 0, false
	}
	defer procCloseHandle.Call(h)

	var pmc processMemCountersEx
	pmc.Cb = uint32(unsafe.Sizeof(pmc))
	r, _, _ := procK32GetProcessMemoryInfo.Call(h, uintptr(unsafe.Pointer(&pmc)), uintptr(pmc.Cb))
	if r == 0 {
		r, _, _ = procPsapiGetProcMemInfo.Call(h, uintptr(unsafe.Pointer(&pmc)), uintptr(pmc.Cb))
	}
	if r == 0 {
		return 0, 0, false
	}
	return uint64(pmc.WorkingSetSize), uint64(pmc.PrivateUsage), true
}

// ListTopProcesses retorna os maiores consumidores de memoria privada sem encerrar nada.
func ListTopProcesses(limit int, selfPID uint32, anomalyLimitMB int) ([]domain.ProcessInfo, error) {
	entries, err := enumSnapshotProcesses()
	if err != nil {
		return nil, err
	}
	fgPID := GetForegroundPID()
	list := make([]domain.ProcessInfo, 0, len(entries))
	for i := range entries {
		pid := entries[i].ProcessID
		if pid <= 4 {
			continue
		}
		ws, priv, ok := readProcMemory(pid)
		if !ok || (ws == 0 && priv == 0) {
			continue
		}
		name := syscall.UTF16ToString(entries[i].ExeFile[:])
		prot := domain.IsProtectedProcess(name, pid, selfPID, fgPID, false)
		list = append(list, domain.ProcessInfo{
			PID:             pid,
			Name:            name,
			PrivateBytes:    priv,
			WorkingSetBytes: ws,
			Protected:       prot,
			Anomaly:         domain.IsAnomalyProcess(priv, anomalyLimitMB),
		})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].PrivateBytes > list[j].PrivateBytes
	})
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

// TrimProcessWorkingSet recicla paginas inativas de um processo sem fecha-lo.
func TrimProcessWorkingSet(pid uint32) error {
	h, _, err := procOpenProcess.Call(procQueryLimitedInfo|procSetQuota, 0, uintptr(pid))
	if h == 0 {
		return fmt.Errorf("OpenProcess(%d): %w", pid, err)
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

// TrimAllSafe aplica reciclagem nao destrutiva em todos os processos nao protegidos.
func TrimAllSafe(selfPID uint32, protectForeground bool) (trimmed int, skipped int) {
	entries, err := enumSnapshotProcesses()
	if err != nil {
		return 0, 0
	}
	fgPID := GetForegroundPID()
	for i := range entries {
		pid := entries[i].ProcessID
		name := syscall.UTF16ToString(entries[i].ExeFile[:])
		if domain.IsProtectedProcess(name, pid, selfPID, fgPID, protectForeground) {
			skipped++
			continue
		}
		if err := TrimProcessWorkingSet(pid); err == nil {
			trimmed++
		} else {
			skipped++
		}
	}
	return trimmed, skipped
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
