//go:build windows

package sys

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const (
	th32csSnapProcess    = 0x00000002
	procTerminate        = 0x0001
	procVMRead           = 0x0010
	procSetQuota         = 0x0100
	procQueryLimitedInfo = 0x1000
	invalidHandleValue   = ^uintptr(0)
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

type fileTime struct{ Low, High uint32 }

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

func readProcDetails(pid uint32, wantCmd bool) (ws, priv uint64, cmd string, ok bool) {
	h, _, _ := procOpenProcess.Call(procQueryLimitedInfo|procVMRead, 0, uintptr(pid))
	if h == 0 {
		h, _, _ = procOpenProcess.Call(procQueryLimitedInfo, 0, uintptr(pid))
	}
	if h == 0 {
		return 0, 0, "", false
	}
	defer procCloseHandle.Call(h)

	var pmc processMemCountersEx
	pmc.Cb = uint32(unsafe.Sizeof(pmc))
	r, _, _ := procK32GetProcessMemoryInfo.Call(h, uintptr(unsafe.Pointer(&pmc)), uintptr(pmc.Cb))
	if r == 0 {
		r, _, _ = procPsapiGetProcMemInfo.Call(h, uintptr(unsafe.Pointer(&pmc)), uintptr(pmc.Cb))
	}
	if r != 0 {
		ws = uint64(pmc.WorkingSetSize)
		priv = uint64(pmc.PrivateUsage)
		ok = true
	}
	if wantCmd {
		cmd = ReadProcessCmdLine(h)
	}
	return ws, priv, cmd, ok
}

func readProcCPUTime(pid uint32) (uint64, bool) {
	h, _, _ := procOpenProcess.Call(procQueryLimitedInfo, 0, uintptr(pid))
	if h == 0 {
		return 0, false
	}
	defer procCloseHandle.Call(h)
	var cr, ex, kt, ut fileTime
	r, _, _ := procGetProcessTimes.Call(h, uintptr(unsafe.Pointer(&cr)), uintptr(unsafe.Pointer(&ex)), uintptr(unsafe.Pointer(&kt)), uintptr(unsafe.Pointer(&ut)))
	if r == 0 {
		return 0, false
	}
	total := (uint64(kt.High)<<32 | uint64(kt.Low)) + (uint64(ut.High)<<32 | uint64(ut.Low))
	return total, true
}

// CollectProcessSnapshots coleta processos, janelas abertas, CmdLine e carga de CPU.
func CollectProcessSnapshots() ([]domain.ProcSnapshot, bool, error) {
	entries, err := enumSnapshotProcesses()
	if err != nil {
		return nil, false, err
	}
	ntSessions := readSessionMapNT()
	openPIDs, deskOK := EnumOpenWindowPIDs()
	netPIDs := ActiveNetworkPIDs()
	snaps := make([]domain.ProcSnapshot, 0, len(entries))
	lsSamples := make(map[uint32]uint64)

	for i := range entries {
		pid := entries[i].ProcessID
		name := syscall.UTF16ToString(entries[i].ExeFile[:])
		norm := strings.ToLower(strings.TrimSpace(name))
		wantCmd := norm == "antigravity.exe" ||
			(strings.HasPrefix(norm, "language_server") && strings.HasSuffix(norm, ".exe")) ||
			norm == "msedgewebview2.exe" || norm == "chrome.exe" || norm == "brave.exe" ||
			norm == "msedge.exe" || norm == "electron.exe"
		ws, priv, cmd, _ := readProcDetails(pid, wantCmd)
		isNetSvc := wantCmd && strings.Contains(strings.ToLower(cmd), "network.mojom.")
		snaps = append(snaps, domain.ProcSnapshot{
			PID:             pid,
			ParentPID:       entries[i].ParentProcessID,
			SessionID:       getSessionID(pid, ntSessions),
			Name:            name,
			CmdLine:         cmd,
			PrivateBytes:    priv,
			WorkingSetBytes: ws,
			HasOpenWindow:   openPIDs[pid],
			HasActiveNet:    netPIDs[pid],
			IsNetService:    isNetSvc,
		})
		if strings.HasPrefix(norm, "language_server") && strings.HasSuffix(norm, ".exe") {
			if ct, ok := readProcCPUTime(pid); ok {
				lsSamples[pid] = ct
			}
		}
	}
	if len(lsSamples) > 0 {
		time.Sleep(40 * time.Millisecond)
		for idx := range snaps {
			if prev, ok := lsSamples[snaps[idx].PID]; ok {
				if cur, ok2 := readProcCPUTime(snaps[idx].PID); ok2 && cur > prev {
					snaps[idx].CPUBusy = true
				}
			}
		}
	}
	return snaps, deskOK, nil
}
