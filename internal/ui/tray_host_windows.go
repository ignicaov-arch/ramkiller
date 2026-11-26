//go:build windows

package ui

import (
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

var procShellNotifyIconW = sys.Shell32.NewProc("Shell_NotifyIconW")

// TrayHost gerencia o ciclo de vida do icone na bandeja e notificacoes nativas.
type TrayHost struct {
	hwnd     uintptr
	uid      uint32
	hIcon    uintptr
	added    bool
	lastTip  string
}

// NewTrayHost inicializa o controlador da bandeja vinculado ao HWND principal.
func NewTrayHost(hwnd uintptr) *TrayHost {
	return &TrayHost{hwnd: hwnd, uid: 1337}
}

func (t *TrayHost) baseData(flags uint32) NOTIFYICONDATAW {
	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = t.hwnd
	nid.UID = t.uid
	nid.UFlags = flags
	nid.UCallbackMessage = WM_TRAYICON
	nid.HIcon = t.hIcon
	return nid
}

// EnsureCreated registra ou recria o icone na barra do Windows (ex.: TaskbarCreated).
func (t *TrayHost) EnsureCreated(hIcon uintptr, tip string) {
	if hIcon != 0 {
		if t.hIcon != 0 && t.hIcon != hIcon {
			procDestroyIcon.Call(t.hIcon)
		}
		t.hIcon = hIcon
	}
	t.lastTip = tip
	nid := t.baseData(NIF_MESSAGE | NIF_ICON | NIF_TIP)
	CopyUTF16(nid.SzTip[:], tip)

	ok, _, _ := procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
	if ok != 0 {
		t.added = true
	}
}

// Update atualiza icone e tooltip sem vazar handles GDI.
func (t *TrayHost) Update(hIcon uintptr, tip string) {
	if !t.added {
		t.EnsureCreated(hIcon, tip)
		return
	}
	flags := uint32(0)
	var oldIcon uintptr
	if hIcon != 0 && hIcon != t.hIcon {
		oldIcon = t.hIcon
		t.hIcon = hIcon
		flags |= NIF_ICON
	}
	if tip != "" && tip != t.lastTip {
		t.lastTip = tip
		flags |= NIF_TIP
	}
	if flags == 0 {
		return
	}
	nid := t.baseData(flags)
	if flags&NIF_TIP != 0 {
		CopyUTF16(nid.SzTip[:], tip)
	}
	procShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
	if oldIcon != 0 {
		procDestroyIcon.Call(oldIcon)
	}
}

// ShowBalloon exibe resultado da limpeza como notificacao nativa da bandeja.
func (t *TrayHost) ShowBalloon(title, body string, sound bool) {
	if !t.added {
		return
	}
	nid := t.baseData(NIF_INFO)
	CopyUTF16(nid.SzInfoTitle[:], title)
	CopyUTF16(nid.SzInfo[:], body)
	nid.DwInfoFlags = NIIF_INFO
	if !sound {
		nid.DwInfoFlags |= NIIF_NOSOUND
	}
	procShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
}

// Destroy remove o icone da bandeja e libera o HICON atual.
func (t *TrayHost) Destroy() {
	if t.added {
		nid := t.baseData(0)
		procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
		t.added = false
	}
	if t.hIcon != 0 {
		procDestroyIcon.Call(t.hIcon)
		t.hIcon = 0
	}
}
