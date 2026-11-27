//go:build windows

package sys

import "unsafe"

const qunsAcceptsNotifications = 5

// IsUserAcceptingNotifications evita notificacoes em jogos, tela cheia, apresentacoes ou Modo Nao Perturbe.
func IsUserAcceptingNotifications() bool {
	if err := procSHQueryUserNotificationState.Find(); err != nil {
		return true
	}
	var state uint32
	hr, _, _ := procSHQueryUserNotificationState.Call(uintptr(unsafe.Pointer(&state)))
	if hr != 0 {
		return true
	}
	return state == qunsAcceptsNotifications
}
