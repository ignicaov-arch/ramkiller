//go:build windows

package sys

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const (
	MainMutexName     = `Local\RamKiller_SingleInstance_v1`
	GuardianMutexName = `Local\RamKiller_Guardian_v1`
	GuardianStopEvent = `Local\RamKiller_StopGuardian_v1`

	syncAccess        = 0x00100000
	eventModifyState  = 0x0002
	waitObject0       = 0x00000000
	errAlreadyExists  = 183
	guardianIntervalMs = 2500
)

// IsMutexHeld verifica se um mutex nomeado ja esta ativo em outro processo.
func IsMutexHeld(name string) bool {
	u16, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return false
	}
	h, _, _ := procOpenMutexW.Call(syncAccess, 0, uintptr(unsafe.Pointer(u16)))
	if h == 0 {
		return false
	}
	procCloseHandle.Call(h)
	return true
}

// SignalGuardianStop solicita encerramento limpo do processo guardiao ativo.
func SignalGuardianStop() {
	u16, err := syscall.UTF16PtrFromString(GuardianStopEvent)
	if err != nil {
		return
	}
	h, _, _ := procOpenEventW.Call(eventModifyState, 0, uintptr(unsafe.Pointer(u16)))
	if h != 0 {
		procSetEvent.Call(h)
		procCloseHandle.Call(h)
	}
}

// EnsureGuardianRunning garante que o processo guardiao de autoregeneracao esteja ativo.
func EnsureGuardianRunning() error {
	if IsPausedForCurrentBoot() || IsMutexHeld(GuardianMutexName) {
		return nil
	}
	exe, err := validatedExePath()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "--guardian")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

// RunGuardian monitora o processo principal e o regenera caso seja encerrado externamente.
func RunGuardian() error {
	if IsPausedForCurrentBoot() {
		return nil
	}
	cfg, _ := LoadConfig()
	if !cfg.SelfHeal {
		return nil
	}

	mPtr, err := syscall.UTF16PtrFromString(GuardianMutexName)
	if err != nil {
		return err
	}
	mHandle, _, lastErr := Kernel32.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(mPtr)))
	if errno, ok := lastErr.(syscall.Errno); ok && errno == errAlreadyExists {
		if mHandle != 0 {
			procCloseHandle.Call(mHandle)
		}
		return nil
	}
	if mHandle == 0 {
		return fmt.Errorf("CreateMutexW guardiao falhou: %v", lastErr)
	}
	defer procCloseHandle.Call(mHandle)

	ePtr, err := syscall.UTF16PtrFromString(GuardianStopEvent)
	if err != nil {
		return err
	}
	stopEvt, _, _ := procCreateEventW.Call(0, 1, 0, uintptr(unsafe.Pointer(ePtr)))
	if stopEvt == 0 {
		return fmt.Errorf("CreateEventW guardiao falhou")
	}
	defer procCloseHandle.Call(stopEvt)
	procResetEvent.Call(stopEvt)
	TrimSelf()

	exe, err := validatedExePath()
	if err != nil {
		return err
	}

	for {
		w, _, _ := procWaitForSingleObject.Call(stopEvt, guardianIntervalMs)
		if uint32(w) == waitObject0 {
			return nil
		}
		if IsPausedForCurrentBoot() {
			return nil
		}
		cfg, _ = LoadConfig()
		if !cfg.SelfHeal {
			return nil
		}
		if !IsMutexHeld(MainMutexName) {
			if IsElevated() {
				cmd := exec.Command(exe, "--minimized")
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				_ = cmd.Start()
			} else if IsSkipUACTaskEnabled() {
				_, _ = runSchtasks("/Run", "/TN", skipUACTask)
			} else {
				_ = RelaunchAsAdmin("--minimized")
			}
		}
		TrimSelf()
	}
}

// SyncGuardianState alinha o estado do guardiao com SelfHeal na configuracao ativa.
func SyncGuardianState(cfg domain.Config) {
	if !cfg.SelfHeal || IsPausedForCurrentBoot() {
		SignalGuardianStop()
		return
	}
	_ = EnsureGuardianRunning()
}
