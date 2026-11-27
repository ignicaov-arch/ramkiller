package domain

import "testing"

func TestClassifyProcessesOpenVsBackgroundAndLanguageServer(t *testing.T) {
	snaps := []ProcSnapshot{
		{PID: 4, ParentPID: 0, SessionID: 0, Name: "System"},
		{PID: 100, ParentPID: 4, SessionID: 1, Name: "explorer.exe"},
		{PID: 200, ParentPID: 100, SessionID: 1, Name: "Antigravity.exe", CmdLine: `"Antigravity.exe"`, HasOpenWindow: true},
		{PID: 201, ParentPID: 200, SessionID: 1, Name: "Antigravity.exe", CmdLine: `"Antigravity.exe" --type=gpu-process`},
		{PID: 202, ParentPID: 200, SessionID: 1, Name: "Antigravity.exe", CmdLine: `"Antigravity.exe" --type=renderer`, PrivateBytes: 3 * gb},
		{PID: 210, ParentPID: 200, SessionID: 1, Name: "language_server.exe", CmdLine: `language_server.exe --subclient_type hub`, PrivateBytes: 900 * mb, CPUBusy: true},
		{PID: 211, ParentPID: 210, SessionID: 1, Name: "ramkiller.exe"},
		{PID: 212, ParentPID: 200, SessionID: 1, Name: "language_server.exe", CmdLine: `language_server.exe --subclient_type hub`, PrivateBytes: 200 * mb},
		{PID: 213, ParentPID: 9999, SessionID: 1, Name: "language_server.exe", CmdLine: `language_server.exe --subclient_type hub`, PrivateBytes: 400 * mb},
		{PID: 214, ParentPID: 200, SessionID: 1, Name: "gopls.exe", PrivateBytes: 350 * mb},
		{PID: 300, ParentPID: 100, SessionID: 1, Name: "chrome.exe", HasOpenWindow: true},
		{PID: 301, ParentPID: 300, SessionID: 1, Name: "chrome.exe", PrivateBytes: 400 * mb},
		{PID: 400, ParentPID: 100, SessionID: 1, Name: "brave.exe", HasOpenWindow: false, PrivateBytes: 250 * mb},
		{PID: 500, ParentPID: 100, SessionID: 1, Name: "msedgewebview2.exe", HasOpenWindow: false, PrivateBytes: 850 * mb},
		{PID: 600, ParentPID: 100, SessionID: 1, Name: "ApplicationFrameHost.exe", HasOpenWindow: true},
		{PID: 601, ParentPID: 600, SessionID: 1, Name: "WhatsApp.Root.exe", HasOpenWindow: true},
		{PID: 700, ParentPID: 100, SessionID: 1, Name: "AyuGram.exe", HasOpenWindow: false},
		{PID: 800, ParentPID: 9888, SessionID: 1, Name: "node.exe", HasOpenWindow: false},
		{PID: 801, ParentPID: 800, SessionID: 1, Name: "electron.exe", HasOpenWindow: false},
		{PID: 802, ParentPID: 801, SessionID: 1, Name: "electron.exe", HasOpenWindow: false},
		{PID: 810, ParentPID: 0, SessionID: 0, Name: "services.exe"},
		{PID: 811, ParentPID: 810, SessionID: 1, Name: "RtkAudUService64.exe"},
		{PID: 900, ParentPID: 100, SessionID: 1, Name: "PredatorSense.exe", HasOpenWindow: false},
		{PID: 950, ParentPID: 100, SessionID: 1, Name: "AnyDesk.exe", HasOpenWindow: false, PrivateBytes: 120 * mb},
		{PID: 951, ParentPID: 950, SessionID: 1, Name: "AnyDesk.exe", CmdLine: `"AnyDesk.exe" --control`, HasOpenWindow: false},
	}

	got := ClassifyProcesses(snaps, 211, true, 2560)

	expect := map[uint32]ProcAction{
		4:   ProcKeepSystem,
		100: ProcKeepSystem,
		200: ProcKeepSystem,  // ancestral do selfPID 211 (via 210 -> 200)
		201: ProcKeepAndTrim, // GPU process Antigravity
		202: ProcKill,        // Renderer Antigravity vazando 3GB
		210: ProcKeepSystem,  // language_server hub ancestral do selfPID 211
		211: ProcKeepSystem,  // proprio RamKiller
		212: ProcKeepAndTrim, // hub PID 212 > 210
		213: ProcKill,        // language_server orfao (pai 9999 morto)
		214: ProcKill,        // gopls secundario sem filho ativo
		300: ProcKeepAndTrim, // Chrome aberto
		301: ProcKeepAndTrim, // Tab filho do Chrome aberto
		400: ProcKill,        // Brave fechado em 2o plano
		500: ProcKill,        // WebView2 orfao sem janela aberta
		601: ProcKeepAndTrim, // WhatsApp
		700: ProcKeepAndTrim, // AyuGram principal
		800: ProcKeepAndTrim, // Node
		801: ProcKeepAndTrim, // Electron filho do Node
		802: ProcKeepAndTrim, // GPU/Renderer filho do Electron gerenciado por Node
		811: ProcKeepSystem,  // Servico filho de services.exe
		900: ProcKill,        // Bloatware
		950: ProcKeepSystem,  // AnyDesk sem titulo Win32 protegido
		951: ProcKeepSystem,  // AnyDesk --control na bandeja protegido
	}

	if got[210].Action != ProcKeepSystem {
		t.Errorf("PID 210 ancestral do self deveria ser KeepSystem, obteve %+v", got[210])
	}
	for pid, want := range expect {
		if got[pid].Action != want {
			t.Errorf("PID %d: esperava %s, obteve %s (%s)", pid, want, got[pid].Action, got[pid].Reason)
		}
	}

	// Caso 2: language_server duplicado e ocioso sem ser ancestral do self
	snaps2 := []ProcSnapshot{
		{PID: 200, ParentPID: 100, SessionID: 1, Name: "Antigravity.exe", CmdLine: `"Antigravity.exe"`, HasOpenWindow: true},
		{PID: 210, ParentPID: 200, SessionID: 1, Name: "language_server.exe", CmdLine: `language_server.exe --subclient_type hub`, PrivateBytes: 3 * gb, CPUBusy: false},
		{PID: 220, ParentPID: 200, SessionID: 1, Name: "language_server.exe", CmdLine: `language_server.exe --subclient_type hub`, PrivateBytes: 700 * mb, CPUBusy: false},
		{PID: 221, ParentPID: 200, SessionID: 1, Name: "language_server.exe", CmdLine: `language_server.exe --worker`, PrivateBytes: 400 * mb},
		{PID: 999, ParentPID: 100, SessionID: 1, Name: "ramkiller.exe"},
	}
	got2 := ClassifyProcesses(snaps2, 999, true, 2560)
	if got2[210].Action != ProcKill {
		t.Errorf("hub antigo/duplicado 210 devia ser morto, obteve %+v", got2[210])
	}
	if got2[220].Action != ProcKeepAndTrim {
		t.Errorf("hub ativo mais novo 220 devia ser preservado e reciclado, obteve %+v", got2[220])
	}
	if got2[221].Action != ProcKill {
		t.Errorf("worker language_server secundario devia ser morto, obteve %+v", got2[221])
	}

	// Caso 3: fail-safe quando desktopEnumOK == false
	gotFailSafe := ClassifyProcesses(snaps, 211, false, 2560)
	if gotFailSafe[400].Action != ProcKeepAndTrim {
		t.Errorf("fail-safe desktopEnumOK=false nao deveria matar app desconhecido (brave.exe): %+v", gotFailSafe[400])
	}
	if gotFailSafe[900].Action != ProcKill {
		t.Errorf("fail-safe ainda deve matar bloatware explicito (PredatorSense): %+v", gotFailSafe[900])
	}
}
