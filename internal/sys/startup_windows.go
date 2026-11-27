//go:build windows

package sys

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unicode"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const (
	hkeyCurrentUser = uintptr(0x80000001)
	keyRead         = 0x20019
	keySetValue     = 0x0002
	regSZ           = 1
	regBinary       = 3
	errFileNotFound = 2

	runSubKey         = `Software\Microsoft\Windows\CurrentVersion\Run`
	runApprovedSubKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	runValueName      = "RamKiller"
)

func hasControlChars(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) || r == '"' {
			return true
		}
	}
	return false
}

func validatedExePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	if exe == "" || hasControlChars(exe) {
		return "", fmt.Errorf("invalid executable path")
	}
	return exe, nil
}

// IsAutorunEnabled verifica se o RamKiller esta registrado no HKCU Run.
func IsAutorunEnabled() bool {
	subKeyPtr, _ := syscall.UTF16PtrFromString(runSubKey)
	valPtr, _ := syscall.UTF16PtrFromString(runValueName)
	var hKey uintptr
	r, _, _ := procRegOpenKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(subKeyPtr)), 0, keyRead, uintptr(unsafe.Pointer(&hKey)))
	if r != 0 || hKey == 0 {
		return false
	}
	defer procRegCloseKey.Call(hKey)

	var valType, dataSize uint32
	r, _, _ = procRegQueryValueExW.Call(hKey, uintptr(unsafe.Pointer(valPtr)), 0, uintptr(unsafe.Pointer(&valType)), 0, uintptr(unsafe.Pointer(&dataSize)))
	return r == 0 && valType == regSZ && dataSize > 0
}

func setStartupApproved(enabled bool) {
	if !enabled {
		return
	}
	subPtr, _ := syscall.UTF16PtrFromString(runApprovedSubKey)
	valPtr, _ := syscall.UTF16PtrFromString(runValueName)
	var hKey uintptr
	r, _, _ := procRegOpenKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(subPtr)), 0, keySetValue, uintptr(unsafe.Pointer(&hKey)))
	if r != 0 || hKey == 0 {
		return
	}
	defer procRegCloseKey.Call(hKey)
	approved := [12]byte{0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	procRegSetValueExW.Call(hKey, uintptr(unsafe.Pointer(valPtr)), 0, regBinary, uintptr(unsafe.Pointer(&approved[0])), uintptr(len(approved)))
}

// SetAutorun registra ou remove o RamKiller em HKCU\...\Run com --boot.
func SetAutorun(enabled bool, minimized bool) error {
	subKeyPtr, _ := syscall.UTF16PtrFromString(runSubKey)
	valPtr, _ := syscall.UTF16PtrFromString(runValueName)
	var hKey uintptr
	r, _, _ := procRegOpenKeyExW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(subKeyPtr)), 0, keySetValue, uintptr(unsafe.Pointer(&hKey)))
	if r != 0 || hKey == 0 {
		return fmt.Errorf("RegOpenKeyExW falhou (%d)", r)
	}
	defer procRegCloseKey.Call(hKey)

	if !enabled {
		r, _, _ = procRegDeleteValueW.Call(hKey, uintptr(unsafe.Pointer(valPtr)))
		if r != 0 && r != errFileNotFound {
			return fmt.Errorf("RegDeleteValueW falhou (%d)", r)
		}
		return nil
	}

	exe, err := validatedExePath()
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf(`"%s" --boot`, exe)
	if minimized {
		cmd = fmt.Sprintf(`"%s" --minimized --boot`, exe)
	}
	u16, err := syscall.UTF16FromString(cmd)
	if err != nil {
		return err
	}
	r, _, _ = procRegSetValueExW.Call(hKey, uintptr(unsafe.Pointer(valPtr)), 0, regSZ, uintptr(unsafe.Pointer(&u16[0])), uintptr(len(u16)*2))
	if r != 0 {
		return fmt.Errorf("RegSetValueExW falhou (%d)", r)
	}
	setStartupApproved(true)
	return nil
}

// SyncStartupRegistration sincroniza Menu Iniciar, Run no registro e tarefas agendadas.
func SyncStartupRegistration(cfg domain.Config) {
	_ = SetAutorun(cfg.LoadOnStartup, cfg.StartMinimized)
	_ = SyncStartMenuShortcut(cfg.StartMenuShortcut)
	if IsElevated() {
		_ = SetSkipUACTask(cfg.LoadOnStartup && cfg.SkipUAC, cfg.StartMinimized)
		_ = SyncInteractiveTask()
	}
}

// OpenTaskManager abre o Gerenciador de Tarefas do Windows a partir de System32.
func OpenTaskManager() error {
	return exec.Command(System32Path("Taskmgr.exe")).Start()
}
