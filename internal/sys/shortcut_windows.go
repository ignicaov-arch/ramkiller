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

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	clsidShellLink = guid{0x00021401, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidShellLinkW  = guid{0x000214F9, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidPersistFile = guid{0x0000010B, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

const (
	coinitApartmentThreaded = 0x2
	clsctxInprocServer      = 0x1
	shcneAssocChanged       = 0x08000000
	shcnfIdList             = 0x0000
)

func comCall(obj unsafe.Pointer, idx int, args ...uintptr) uintptr {
	vtbl := (*[24]uintptr)(*(*unsafe.Pointer)(obj))
	params := make([]uintptr, 0, 1+len(args))
	params = append(params, uintptr(obj))
	params = append(params, args...)
	r, _, _ := syscall.SyscallN(vtbl[idx], params...)
	return r
}

func checkHResult(hr uintptr, op string) error {
	if int32(hr) < 0 {
		return fmt.Errorf("%s falhou: HRESULT 0x%08X", op, uint32(hr))
	}
	return nil
}

// StartMenuShortcutPath retorna o caminho canonico do atalho no Menu Iniciar do usuario.
func StartMenuShortcutPath() (string, error) {
	appData := strings.TrimSpace(os.Getenv("APPDATA"))
	if appData == "" {
		return "", fmt.Errorf("APPDATA indisponivel")
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "RamKiller.lnk"), nil
}

// IsStartMenuShortcutPresent verifica se RamKiller.lnk existe no Menu Iniciar.
func IsStartMenuShortcutPresent() bool {
	p, err := StartMenuShortcutPath()
	if err != nil {
		return false
	}
	info, statErr := os.Stat(p)
	return statErr == nil && !info.IsDir()
}

// CreateShortcutLNK grava um arquivo .lnk nativo via COM IShellLinkW + IPersistFile.
func CreateShortcutLNK(lnkPath, targetExe, args, workDir, desc, iconPath string, iconIdx int) error {
	if err := os.MkdirAll(filepath.Dir(lnkPath), 0o755); err != nil {
		return fmt.Errorf("mkdir atalho: %w", err)
	}
	hrInit, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if hrInit == 0 || hrInit == 1 {
		defer procCoUninitialize.Call()
	} else if int32(hrInit) < 0 && uint32(hrInit) != 0x80010106 {
		return fmt.Errorf("CoInitializeEx falhou: 0x%08X", uint32(hrInit))
	}

	var shellObj unsafe.Pointer
	hr, _, _ := procCoCreateInst.Call(
		uintptr(unsafe.Pointer(&clsidShellLink)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidShellLinkW)),
		uintptr(unsafe.Pointer(&shellObj)),
	)
	if err := checkHResult(hr, "CoCreateInstance(ShellLink)"); err != nil {
		return err
	}
	defer comCall(shellObj, 2)

	u16 := func(s string) uintptr {
		p, _ := syscall.UTF16PtrFromString(s)
		return uintptr(unsafe.Pointer(p))
	}
	if err := checkHResult(comCall(shellObj, 20, u16(targetExe)), "IShellLinkW.SetPath"); err != nil {
		return err
	}
	if err := checkHResult(comCall(shellObj, 11, u16(args)), "IShellLinkW.SetArguments"); err != nil {
		return err
	}
	if err := checkHResult(comCall(shellObj, 9, u16(workDir)), "IShellLinkW.SetWorkingDirectory"); err != nil {
		return err
	}
	if err := checkHResult(comCall(shellObj, 7, u16(desc)), "IShellLinkW.SetDescription"); err != nil {
		return err
	}
	if iconPath != "" {
		if err := checkHResult(comCall(shellObj, 17, u16(iconPath), uintptr(iconIdx)), "IShellLinkW.SetIconLocation"); err != nil {
			return err
		}
	}

	var persistObj unsafe.Pointer
	hrQI := comCall(shellObj, 0, uintptr(unsafe.Pointer(&iidPersistFile)), uintptr(unsafe.Pointer(&persistObj)))
	if err := checkHResult(hrQI, "QueryInterface(IPersistFile)"); err != nil {
		return err
	}
	defer comCall(persistObj, 2)

	if err := checkHResult(comCall(persistObj, 6, u16(lnkPath), 1), "IPersistFile.Save"); err != nil {
		return err
	}
	procSHChangeNotify.Call(shcneAssocChanged, shcnfIdList, 0, 0)
	return nil
}

// SyncStartMenuShortcut cria, atualiza ou remove RamKiller.lnk no Menu Iniciar do Windows.
func SyncStartMenuShortcut(enabled bool) error {
	lnkPath, err := StartMenuShortcutPath()
	if err != nil {
		return err
	}
	if !enabled {
		if rmErr := os.Remove(lnkPath); rmErr != nil && !os.IsNotExist(rmErr) {
			return rmErr
		}
		procSHChangeNotify.Call(shcneAssocChanged, shcnfIdList, 0, 0)
		return nil
	}
	exe, err := validatedExePath()
	if err != nil {
		return err
	}
	return CreateShortcutLNK(
		lnkPath,
		exe,
		"",
		filepath.Dir(exe),
		"RamKiller - Limpeza real de RAM e Paginacao no Windows",
		exe,
		0,
	)
}
