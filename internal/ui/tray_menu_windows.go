//go:build windows

package ui

import (
	"fmt"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

const (
	cmdTrayToggle        = 1001
	cmdTrayClean         = 1002
	cmdTrayCleanPagefile = 1003
	cmdTraySettings      = 1004
	cmdTrayExit          = 1005
	cmdTrayToggleKillBg  = 1006
	cmdTrayPauseReboot   = 1007

	cmdRegionBase   = 1100
	cmdThreshOff    = 1200
	cmdThreshBase   = 1201
	cmdIntervalOff  = 1300
	cmdIntervalBase = 1301
)

var (
	thresholdOptions = []int{50, 60, 70, 75, 80, 85, 90, 95}
	intervalOptions  = []int{1, 5, 10, 15, 30, 60, 120}

	procCreatePopupMenu  = sys.User32.NewProc("CreatePopupMenu")
	procAppendMenuW      = sys.User32.NewProc("AppendMenuW")
	procTrackPopupMenuEx = sys.User32.NewProc("TrackPopupMenuEx")
	procDestroyMenu      = sys.User32.NewProc("DestroyMenu")
	procGetCursorPos     = sys.User32.NewProc("GetCursorPos")
	procSetForegroundWnd = sys.User32.NewProc("SetForegroundWindow")
	procPostMessageW     = sys.User32.NewProc("PostMessageW")
)

func addMenuItem(hMenu, flags, id uintptr, label string) {
	procAppendMenuW.Call(hMenu, flags, id, uintptr(unsafe.Pointer(UTF16Ptr(label))))
}

func addMenuSep(hMenu uintptr) {
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
}

func buildRegionsSubmenu(cfg domain.Config) uintptr {
	sub, _, _ := procCreatePopupMenu.Call()
	specs := domain.AllRegionSpecs()
	for i, s := range specs {
		flags := uintptr(MF_STRING)
		if cfg.CleanMask&s.Flag == s.Flag {
			flags |= MF_CHECKED
		}
		addMenuItem(sub, flags, uintptr(cmdRegionBase+i), s.Label)
	}
	return sub
}

func buildThresholdSubmenu(cfg domain.Config) uintptr {
	sub, _, _ := procCreatePopupMenu.Call()
	offFlags := uintptr(MF_STRING)
	if !cfg.AutoCleanThresholdEnabled {
		offFlags |= MF_CHECKED
	}
	addMenuItem(sub, offFlags, cmdThreshOff, "Desativar")
	addMenuSep(sub)
	for i, v := range thresholdOptions {
		flags := uintptr(MF_STRING)
		if cfg.AutoCleanThresholdEnabled && cfg.AutoCleanThresholdPct == v {
			flags |= MF_CHECKED
		}
		addMenuItem(sub, flags, uintptr(cmdThreshBase+i), fmt.Sprintf("%d%%", v))
	}
	return sub
}

func buildIntervalSubmenu(cfg domain.Config) uintptr {
	sub, _, _ := procCreatePopupMenu.Call()
	offFlags := uintptr(MF_STRING)
	if !cfg.AutoCleanIntervalEnabled {
		offFlags |= MF_CHECKED
	}
	addMenuItem(sub, offFlags, cmdIntervalOff, "Desativar")
	addMenuSep(sub)
	for i, m := range intervalOptions {
		flags := uintptr(MF_STRING)
		if cfg.AutoCleanIntervalEnabled && cfg.AutoCleanIntervalMin == m {
			flags |= MF_CHECKED
		}
		addMenuItem(sub, flags, uintptr(cmdIntervalBase+i), fmt.Sprintf("%d min", m))
	}
	return sub
}

// ShowTrayContextMenu exibe o menu de contexto completo em PT-BR com todas as acoes.
func ShowTrayContextMenu(hwnd uintptr, cfg domain.Config) uint32 {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return 0
	}
	defer procDestroyMenu.Call(hMenu)

	subReg := buildRegionsSubmenu(cfg)
	subThr := buildThresholdSubmenu(cfg)
	subInt := buildIntervalSubmenu(cfg)

	addMenuItem(hMenu, MF_STRING, cmdTrayToggle, "Abrir / Ocultar Painel")
	addMenuSep(hMenu)
	addMenuItem(hMenu, MF_STRING, cmdTrayClean, "Limpar Memória")
	addMenuItem(hMenu, MF_STRING, cmdTrayCleanPagefile, "Limpar Paginação")
	addMenuSep(hMenu)
	addMenuItem(hMenu, MF_POPUP, subReg, "Áreas de Limpeza")
	killFlags := uintptr(MF_STRING)
	if cfg.KillBackgroundApps {
		killFlags |= MF_CHECKED
	}
	addMenuItem(hMenu, killFlags, cmdTrayToggleKillBg, "Matar 2º Plano ao Limpar")
	addMenuItem(hMenu, MF_POPUP, subThr, "Limpar Acima de")
	addMenuItem(hMenu, MF_POPUP, subInt, "Limpar a Cada")
	addMenuSep(hMenu)
	addMenuItem(hMenu, MF_STRING, cmdTraySettings, "Configurações...")
	addMenuSep(hMenu)
	addMenuItem(hMenu, MF_STRING, cmdTrayPauseReboot, "Encerrar até Reiniciar o PC")
	addMenuItem(hMenu, MF_STRING, cmdTrayExit, "Sair")

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWnd.Call(hwnd)
	cmd, _, _ := procTrackPopupMenuEx.Call(hMenu, TPM_RIGHTBUTTON|TPM_RETURNCMD, uintptr(pt.X), uintptr(pt.Y), hwnd, 0)
	procPostMessageW.Call(hwnd, 0, 0, 0)
	return uint32(cmd)
}
