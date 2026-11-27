package domain

import "testing"

func TestDescribeProcessAndBloatwareWebViewKill(t *testing.T) {
	snaps := []ProcSnapshot{
		{PID: 31596, ParentPID: 100, SessionID: 1, Name: "WhatsApp.Root.exe", HasOpenWindow: true},
		{PID: 38028, ParentPID: 31596, SessionID: 1, Name: "msedgewebview2.exe", CmdLine: `"msedgewebview2.exe" --embedded-browser-webview=1 --webview-exe-name=WhatsApp.Root.exe`},
		{PID: 2784, ParentPID: 38028, SessionID: 1, Name: "msedgewebview2.exe", CmdLine: `"msedgewebview2.exe" --type=renderer --webview-exe-name=WhatsApp.Root.exe`},
		{PID: 68312, ParentPID: 100, SessionID: 1, Name: "SearchHost.exe", HasOpenWindow: false},
		{PID: 35212, ParentPID: 68312, SessionID: 1, Name: "msedgewebview2.exe", CmdLine: `"msedgewebview2.exe" --embedded-browser-webview=1 --webview-exe-name=SearchHost.exe`},
		{PID: 68756, ParentPID: 35212, SessionID: 1, Name: "msedgewebview2.exe", IsNetService: true, CmdLine: `"msedgewebview2.exe" --type=utility --utility-sub-type=network.mojom.NetworkService --webview-exe-name=SearchHost.exe`},
		{PID: 67972, ParentPID: 100, SessionID: 1, Name: "TabTip.exe", HasOpenWindow: false},
		{PID: 59584, ParentPID: 100, SessionID: 1, Name: "language_server.exe", HasOpenWindow: true},
	}

	byPID := make(map[uint32]ProcSnapshot, len(snaps))
	for _, s := range snaps {
		byPID[s.PID] = s
	}
	decs := ClassifyProcesses(snaps, 99999, true, 2560)

	if decs[2784].Action != ProcKeepAndTrim {
		t.Errorf("WhatsApp renderer 2784 deveria estar protegido, obteve %+v", decs[2784])
	}
	if decs[68312].Action != ProcKill || decs[35212].Action != ProcKill || decs[68756].Action != ProcKill {
		t.Errorf("SearchHost fechado e seu WebView2 deveriam virar Matar: host=%+v wv=%+v net=%+v", decs[68312], decs[35212], decs[68756])
	}
	if decs[67972].Action != ProcKill {
		t.Errorf("TabTip ocioso deveria ser Matar, obteve %+v", decs[67972])
	}

	cases := map[uint32]string{
		2784:  "WhatsApp · Interface / Renderização",
		35212: "Busca Web Windows · Motor Principal WebView2",
		67972: "Teclado Virtual / Emojis (TIP)",
		59584: "Antigravity · Motor de IA e Código",
	}
	for pid, want := range cases {
		if got := DescribeProcess(byPID[pid], byPID, decs[pid]); got != want {
			t.Errorf("PID %d: esperava %q, obteve %q", pid, want, got)
		}
	}
}
