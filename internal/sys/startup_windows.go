//go:build windows

package sys

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode"
	"unsafe"
)

const (
	hkeyCurrentUser = uintptr(0x80000001)
	keyRead         = 0x20019
	keySetValue     = 0x0002
	regSZ           = 1
	errFileNotFound = 2

	runSubKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName = "RamKiller"
	skipUACTask  = "RamKiller_SkipUAC"
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

// IsAutorunEnabled checks whether RamKiller is registered in HKCU Run.
func IsAutorunEnabled() bool {
	subKeyPtr, err := syscall.UTF16PtrFromString(runSubKey)
	if err != nil {
		return false
	}
	valPtr, err := syscall.UTF16PtrFromString(runValueName)
	if err != nil {
		return false
	}

	var hKey uintptr
	r, _, _ := procRegOpenKeyExW.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(subKeyPtr)),
		0,
		keyRead,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if r != 0 || hKey == 0 {
		return false
	}
	defer procRegCloseKey.Call(hKey)

	var valType uint32
	var dataSize uint32
	r, _, _ = procRegQueryValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valPtr)),
		0,
		uintptr(unsafe.Pointer(&valType)),
		0,
		uintptr(unsafe.Pointer(&dataSize)),
	)
	return r == 0 && valType == regSZ && dataSize > 0
}

// SetAutorun registers or removes RamKiller from HKCU\...\Run.
func SetAutorun(enabled bool, minimized bool) error {
	subKeyPtr, err := syscall.UTF16PtrFromString(runSubKey)
	if err != nil {
		return err
	}
	valPtr, err := syscall.UTF16PtrFromString(runValueName)
	if err != nil {
		return err
	}

	var hKey uintptr
	r, _, _ := procRegOpenKeyExW.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(subKeyPtr)),
		0,
		keySetValue,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if r != 0 || hKey == 0 {
		return fmt.Errorf("RegOpenKeyExW failed with code %d", r)
	}
	defer procRegCloseKey.Call(hKey)

	if !enabled {
		r, _, _ = procRegDeleteValueW.Call(hKey, uintptr(unsafe.Pointer(valPtr)))
		if r != 0 && r != errFileNotFound {
			return fmt.Errorf("RegDeleteValueW failed with code %d", r)
		}
		return nil
	}

	exe, err := validatedExePath()
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf(`"%s"`, exe)
	if minimized {
		cmd += " --minimized"
	}
	u16, err := syscall.UTF16FromString(cmd)
	if err != nil {
		return err
	}
	byteLen := uint32(len(u16) * 2)
	r, _, _ = procRegSetValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valPtr)),
		0,
		regSZ,
		uintptr(unsafe.Pointer(&u16[0])),
		uintptr(byteLen),
	)
	if r != 0 {
		return fmt.Errorf("RegSetValueExW failed with code %d", r)
	}
	return nil
}

func runSchtasks(args ...string) (string, error) {
	cmd := exec.Command(System32Path("schtasks.exe"), args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// IsSkipUACTaskEnabled reports whether the RamKiller_SkipUAC scheduled task exists.
func IsSkipUACTaskEnabled() bool {
	_, err := runSchtasks("/Query", "/TN", skipUACTask)
	return err == nil
}

// SetSkipUACTask creates or removes the highest-privilege logon task to skip UAC prompts.
func SetSkipUACTask(enabled bool) error {
	if !enabled {
		out, err := runSchtasks("/Delete", "/TN", skipUACTask, "/F")
		if err != nil && IsSkipUACTaskEnabled() {
			return fmt.Errorf("schtasks delete failed: %v (%s)", err, out)
		}
		return nil
	}

	exe, err := validatedExePath()
	if err != nil {
		return err
	}
	tr := fmt.Sprintf(`"%s"`, exe)
	out, err := runSchtasks(
		"/Create",
		"/TN", skipUACTask,
		"/TR", tr,
		"/SC", "ONLOGON",
		"/RL", "HIGHEST",
		"/F",
	)
	if err != nil {
		return fmt.Errorf("schtasks create failed: %v (%s)", err, out)
	}
	return nil
}

// OpenTaskManager launches Windows Task Manager from System32.
func OpenTaskManager() error {
	cmd := exec.Command(System32Path("Taskmgr.exe"))
	return cmd.Start()
}
