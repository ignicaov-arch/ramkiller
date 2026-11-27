//go:build windows

package ui

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
	"github.com/ignicaov-arch/ramkiller/internal/sys"
)

const errorAlreadyExists = 183

// AppController orquestra estado, janela principal, configuracoes, bandeja, hotkey e guardiao.
type AppController struct {
	mu                sync.RWMutex
	cfg               domain.Config
	stats             domain.MemoryStats
	engine            *sys.Engine
	tray              *TrayHost
	mainHWND           uintptr
	settingsHWND       uintptr
	mainHits           []HitRegion
	settingsHits       []HitRegion
	mainHover          string
	settingsHover      string
	settingsTab        int
	lastSummary        string
	startedUnix        int64
	lastCleanUnix      int64
	lastIconPct        int
	tickCount          uint64
	pendingResult      *domain.CleanResult
	taskbarCreatedMsg  uint32
	procScrollY        int
	procMaxScroll      int
	scrollTrack        RECT
	scrollThumb        RECT
	scrollDragging     bool
	scrollDragStartY   int32
	scrollDragStartVal int
}

// FocusRunningInstance traz a janela do RamKiller ja em execucao para frente sem exigir novo UAC.
func FocusRunningInstance() bool {
	existing, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(UTF16Ptr(mainClassName))), 0)
	if existing == 0 {
		return false
	}
	procAllowSetForegroundWindow.Call(^uintptr(0))
	procPostMessageW.Call(existing, wmAppShow, 0, 0)
	return true
}

// RunApp inicializa o aplicativo nativo na bandeja do Windows (loop Win32 em thread fixa).
func RunApp(startMinimized bool, isBoot bool) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	mutexHandle, _, lastErr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(UTF16Ptr(sys.MainMutexName))))
	if errno, ok := lastErr.(syscall.Errno); ok && errno == errorAlreadyExists {
		if existing, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(UTF16Ptr(mainClassName))), 0); existing != 0 {
			procPostMessageW.Call(existing, wmAppShow, 0, 0)
		}
		return nil
	}
	if mutexHandle != 0 {
		defer closeWinHandle(mutexHandle)
	}

	cfg, _ := sys.LoadConfig()
	cfg.Sanitize()
	if sys.IsElevated() {
		_ = sys.EnableMemoryPrivileges()
	}
	sys.SyncStartupRegistration(cfg)
	_ = sys.SaveConfig(cfg)
	if cfg.SelfHeal {
		_ = sys.EnsureGuardianRunning()
	}
	if isBoot && cfg.StartMinimized {
		startMinimized = true
	}

	nowUnix := time.Now().Unix()
	ctrl := &AppController{
		cfg:         cfg,
		engine:      sys.NewEngine(),
		startedUnix: nowUnix,
		lastIconPct: -1,
	}
	ctrl.taskbarCreatedMsg = registerTaskbarMsg()
	ctrl.refreshStats(!startMinimized)

	hwnd := createFixedDarkWindow(mainClassName, "RamKiller", mainWinWidth, mainWinHeight, syscall.NewCallback(ctrl.mainWndProc), !startMinimized)
	if hwnd == 0 {
		return fmt.Errorf("falha ao criar janela principal")
	}
	ctrl.mainHWND = hwnd
	ApplyTopMost(hwnd, cfg.AlwaysOnTop)

	ctrl.tray = NewTrayHost(hwnd)
	ctrl.forceTrayRefresh()
	ctrl.registerHotkey()

	procSetTimer.Call(hwnd, 1, 1000, 0)
	if startMinimized {
		sys.TrimSelf()
	}

	var msg MSG
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	return nil
}

func closeWinHandle(h uintptr) {
	sys.Kernel32.NewProc("CloseHandle").Call(h)
}

func registerTaskbarMsg() uint32 {
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(UTF16Ptr("TaskbarCreated"))))
	return uint32(r)
}

func (a *AppController) getConfig() domain.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

func (a *AppController) applyNewConfig(cfg domain.Config) {
	cfg.Sanitize()
	a.mu.Lock()
	prev := a.cfg
	a.cfg = cfg
	a.lastIconPct = -1
	if cfg.AutoCleanIntervalEnabled && (!prev.AutoCleanIntervalEnabled || prev.AutoCleanIntervalMin != cfg.AutoCleanIntervalMin) && a.lastCleanUnix == 0 {
		a.startedUnix = time.Now().Unix()
	}
	a.mu.Unlock()
	_ = sys.SaveConfig(cfg)

	if prev.LoadOnStartup != cfg.LoadOnStartup || prev.StartMenuShortcut != cfg.StartMenuShortcut || prev.StartMinimized != cfg.StartMinimized || prev.SkipUAC != cfg.SkipUAC {
		sys.SyncStartupRegistration(cfg)
	}
	if prev.SelfHeal != cfg.SelfHeal {
		sys.SyncGuardianState(cfg)
	}

	ApplyTopMost(a.mainHWND, cfg.AlwaysOnTop)
	if a.settingsHWND != 0 {
		ApplyTopMost(a.settingsHWND, cfg.AlwaysOnTop)
	}
	a.registerHotkey()
	a.forceTrayRefresh()
	procInvalidateRect.Call(a.mainHWND, 0, 0)
	if a.settingsHWND != 0 {
		procInvalidateRect.Call(a.settingsHWND, 0, 0)
	}
}

func (a *AppController) registerHotkey() {
	procUnregisterHotKey.Call(a.mainHWND, 1)
	cfg := a.getConfig()
	if cfg.HotkeyEnabled && cfg.HotkeyKey != 0 {
		procRegisterHotKey.Call(a.mainHWND, 1, uintptr(cfg.HotkeyModifiers), uintptr(cfg.HotkeyKey))
	}
}
