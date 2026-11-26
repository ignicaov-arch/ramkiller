//go:build windows

package ui

import (
	"strings"
	"syscall"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const settingsClassName = "RamKillerSettingsClass"

func (a *AppController) openSettingsWindow() {
	if a.settingsHWND != 0 {
		procShowWindow.Call(a.settingsHWND, SW_RESTORE)
		procSetForegroundWnd.Call(a.settingsHWND)
		return
	}
	hwnd := createFixedDarkWindow(settingsClassName, "Configurações · RamKiller", settingsWinWidth, settingsWinHeight, syscall.NewCallback(a.settingsWndProc), false)
	if hwnd != 0 {
		a.settingsHWND = hwnd
		procShowWindow.Call(hwnd, SW_SHOWNORMAL)
		procUpdateWindow.Call(hwnd)
	}
}

func (a *AppController) settingsWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			cfg := a.getConfig()
			canvas, hits := RenderSettingsCanvas(a.settingsTab, cfg, a.stats.Elevated, a.settingsHover)
			a.settingsHits = hits
			canvas.BlitTo(hdc)
			canvas.Close()
			procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		}
		return 0
	case WM_MOUSEMOVE:
		x := int32(int16(lParam & 0xFFFF))
		y := int32(int16((lParam >> 16) & 0xFFFF))
		if hit, ok := FindHit(a.settingsHits, x, y); ok {
			if a.settingsHover != hit.ID {
				a.settingsHover = hit.ID
				procInvalidateRect.Call(hwnd, 0, 0)
			}
			cur, _, _ := procLoadCursorW.Call(0, IDC_HAND)
			procSetCursor.Call(cur)
		} else if a.settingsHover != "" {
			a.settingsHover = ""
			procInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case WM_LBUTTONUP:
		x := int32(int16(lParam & 0xFFFF))
		y := int32(int16((lParam >> 16) & 0xFFFF))
		if hit, ok := FindHit(a.settingsHits, x, y); ok {
			a.handleSettingsClick(hwnd, hit)
		}
		return 0
	case WM_CLOSE:
		procShowWindow.Call(hwnd, SW_HIDE)
		return 0
	case WM_DESTROY:
		a.settingsHWND = 0
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return ret
}

func (a *AppController) handleSettingsClick(hwnd uintptr, hit HitRegion) {
	if strings.HasPrefix(hit.ID, "tab_") {
		a.settingsTab = int(hit.ID[4] - '0')
		procInvalidateRect.Call(hwnd, 0, 0)
		return
	}
	if hit.ID == "settings_close" {
		procShowWindow.Call(hwnd, SW_HIDE)
		return
	}
	if hit.ID == "settings_reset" {
		if ConfirmBox(hwnd, "Restaurar Padrão", "Deseja restaurar todas as configurações do RamKiller para o padrão?") {
			a.applyNewConfig(domain.DefaultConfig())
		}
		return
	}

	cfg := a.getConfig()
	if strings.HasPrefix(hit.ID, "reg_") {
		flag := domain.CleanRegion(hit.Arg)
		if flag&domain.MaskFreezeRisk != 0 && cfg.CleanMask&flag == 0 {
			if !ConfirmBox(hwnd, "Aviso de Estabilidade", "Ativar Lista de Espera* ou Páginas Modificadas* pode causar micro-travamento durante a limpeza manual. Deseja ativar?") {
				return
			}
		}
		cfg.CleanMask ^= flag
		a.applyNewConfig(cfg)
		return
	}

	a.applySettingsMutation(hwnd, hit.ID, &cfg)
	a.applyNewConfig(cfg)
}
