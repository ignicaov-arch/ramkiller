//go:build windows

package sys

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const (
	tokenQuery            = 0x0008
	tokenAdjustPrivileges = 0x0020
	tokenElevationClass   = 20
	sePrivilegeEnabled    = 0x00000002
	swShowNormal          = 1
)

type luid struct {
	LowPart  uint32
	HighPart int32
}

type luidAndAttributes struct {
	Luid       luid
	Attributes uint32
}

type tokenPrivileges struct {
	PrivilegeCount uint32
	Privileges     [1]luidAndAttributes
}

// IsElevated reports whether the current process has an elevated administrator token.
func IsElevated() bool {
	hProc, _, _ := procGetCurrentProcess.Call()
	var token syscall.Handle
	r, _, _ := procOpenProcessToken.Call(
		hProc,
		tokenQuery,
		uintptr(unsafe.Pointer(&token)),
	)
	if r == 0 || token == 0 {
		return false
	}
	defer procCloseHandle.Call(uintptr(token))

	var elevation uint32
	var retLen uint32
	r, _, _ = procGetTokenInformation.Call(
		uintptr(token),
		tokenElevationClass,
		uintptr(unsafe.Pointer(&elevation)),
		unsafe.Sizeof(elevation),
		uintptr(unsafe.Pointer(&retLen)),
	)
	return r != 0 && elevation != 0
}

// EnableMemoryPrivileges attempts to enable memory management privileges on the process token.
func EnableMemoryPrivileges() error {
	hProc, _, _ := procGetCurrentProcess.Call()
	var token syscall.Handle
	r, _, err := procOpenProcessToken.Call(
		hProc,
		tokenAdjustPrivileges|tokenQuery,
		uintptr(unsafe.Pointer(&token)),
	)
	if r == 0 || token == 0 {
		return fmt.Errorf("OpenProcessToken failed: %w", err)
	}
	defer procCloseHandle.Call(uintptr(token))

	privs := []string{
		"SeProfileSingleProcessPrivilege",
		"SeIncreaseQuotaPrivilege",
		"SeDebugPrivilege",
	}
	for _, name := range privs {
		namePtr, convErr := syscall.UTF16PtrFromString(name)
		if convErr != nil {
			continue
		}
		var id luid
		ok, _, _ := procLookupPrivilegeValueW.Call(
			0,
			uintptr(unsafe.Pointer(namePtr)),
			uintptr(unsafe.Pointer(&id)),
		)
		if ok == 0 {
			continue
		}
		tp := tokenPrivileges{
			PrivilegeCount: 1,
			Privileges: [1]luidAndAttributes{
				{Luid: id, Attributes: sePrivilegeEnabled},
			},
		}
		procAdjustTokenPrivileges.Call(
			uintptr(token),
			0,
			uintptr(unsafe.Pointer(&tp)),
			unsafe.Sizeof(tp),
			0,
			0,
		)
	}
	return nil
}

// RelaunchAsAdmin restarts the current executable with UAC elevation via ShellExecuteW("runas").
func RelaunchAsAdmin(args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	verbPtr, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	exePtr, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	dirPtr, err := syscall.UTF16PtrFromString(filepath.Dir(exe))
	if err != nil {
		return err
	}

	var paramsPtr *uint16
	if len(args) > 0 {
		escaped := make([]string, len(args))
		for i, a := range args {
			escaped[i] = syscall.EscapeArg(a)
		}
		paramsPtr, err = syscall.UTF16PtrFromString(strings.Join(escaped, " "))
		if err != nil {
			return err
		}
	}

	r, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(paramsPtr)),
		uintptr(unsafe.Pointer(dirPtr)),
		swShowNormal,
	)
	if r <= 32 {
		return fmt.Errorf("ShellExecuteW runas failed (code %d): %w", r, callErr)
	}
	return nil
}
