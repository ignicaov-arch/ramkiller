package domain

import "strings"

var essentialAppNames = map[string]struct{}{
	"whatsapp.exe":           {},
	"whatsapp.root.exe":      {},
	"ayugram.exe":            {},
	"telegram.exe":           {},
	"discord.exe":            {},
	"leagueclient.exe":       {},
	"leagueclientux.exe":     {},
	"league of legends.exe":  {},
	"riotclientservices.exe": {},
	"riotclientux.exe":       {},
	"node.exe":               {},
	"memreduct.exe":          {},
	"wsl.exe":                {},
	"wslservice.exe":         {},
	"tailscale-ipn.exe":      {},
	"tailscale.exe":          {},
	"cloudflare warp.exe":    {},
	"nordvpn.exe":            {},
}

var standaloneLSPNames = map[string]struct{}{
	"gopls.exe":                      {},
	"rust-analyzer.exe":              {},
	"rust-analyzer-proc-macro.exe":   {},
	"pyright-langserver.exe":         {},
	"pylance.exe":                    {},
	"clangd.exe":                     {},
	"typescript-language-server.exe": {},
}

var activeTaskChildNames = map[string]struct{}{
	"powershell.exe": {},
	"pwsh.exe":       {},
	"cmd.exe":        {},
	"git.exe":        {},
	"go.exe":         {},
	"python.exe":     {},
	"python3.exe":    {},
}

var bloatwareNames = map[string]struct{}{
	"predatorsense.exe":         {},
	"tabtip.exe":                {},
	"onedrive.sync.service.exe": {},
	"crossdeviceresume.exe":     {},
	"backgroundtaskhost.exe":    {},
	"lockapp.exe":               {},
	"igcc.exe":                  {},
	"widgets.exe":               {},
	"widgetservice.exe":         {},
	"searchhost.exe":            {},
	"googleupdate.exe":          {},
	"microsoftedgeupdate.exe":   {},
	"braveupdate.exe":           {},
}

func classifyAntigravity(s ProcSnapshot, brokers map[uint32]bool, openTree map[uint32]bool, anomalyLimitMB int) ProcDecision {
	if len(brokers) == 0 {
		return ProcDecision{Action: ProcKill, Reason: "antigravity em segundo plano sem janela principal"}
	}
	cmd := strings.ToLower(s.CmdLine)
	if strings.Contains(cmd, "--type=renderer") && !s.HasOpenWindow && IsAnomalyProcess(s.PrivateBytes, anomalyLimitMB) {
		return ProcDecision{Action: ProcKill, Reason: "renderer Antigravity com vazamento anômalo"}
	}
	if s.ParentPID > 4 && !brokers[s.ParentPID] && !openTree[s.PID] && !brokers[s.PID] && strings.Contains(cmd, "--type=") {
		return ProcDecision{Action: ProcKill, Reason: "subprocesso Antigravity órfão"}
	}
	return ProcDecision{Action: ProcKeepAndTrim, Reason: "Antigravity ativo protegido e reciclado"}
}

func classifyLanguageServer(
	s ProcSnapshot,
	brokers map[uint32]bool,
	bestHub map[uint32]uint32,
	children map[uint32][]uint32,
	byPID map[uint32]ProcSnapshot,
	anomalyLimitMB int,
) ProcDecision {
	if !brokers[s.ParentPID] {
		return ProcDecision{Action: ProcKill, Reason: "language_server órfão (Antigravity pai encerrado)"}
	}
	cmd := strings.ToLower(s.CmdLine)
	if cmd != "" && !strings.Contains(cmd, "--subclient_type hub") && !hasActiveTaskChild(s.PID, children, byPID) {
		return ProcDecision{Action: ProcKill, Reason: "worker language_server secundário"}
	}
	if keepPID := bestHub[s.ParentPID]; keepPID != 0 && keepPID != s.PID && !hasActiveTaskChild(s.PID, children, byPID) {
		return ProcDecision{Action: ProcKill, Reason: "language_server duplicado obsoleto"}
	}
	if IsAnomalyProcess(s.PrivateBytes, anomalyLimitMB) && !s.CPUBusy && !hasActiveTaskChild(s.PID, children, byPID) {
		return ProcDecision{Action: ProcKill, Reason: "language_server ocioso com vazamento anômalo"}
	}
	return ProcDecision{Action: ProcKeepAndTrim, Reason: "language_server ativo reciclado sem fechar"}
}

func decideSingleProc(
	s ProcSnapshot,
	norm string,
	byPID map[uint32]ProcSnapshot,
	children map[uint32][]uint32,
	brokers map[uint32]bool,
	selfAncestors map[uint32]bool,
	svcTree map[uint32]bool,
	openTree map[uint32]bool,
	essentialTree map[uint32]bool,
	netTree map[uint32]bool,
	bestHub map[uint32]uint32,
	desktopEnumOK bool,
	anomalyLimitMB int,
) ProcDecision {
	if s.PID <= 4 || svcTree[s.PID] || selfAncestors[s.PID] {
		return ProcDecision{Action: ProcKeepSystem, Reason: "núcleo do sistema / serviço Windows"}
	}
	if _, ok := protectedNames[norm]; ok {
		return ProcDecision{Action: ProcKeepSystem, Reason: "processo protegido do Windows"}
	}
	if norm == "conhost.exe" {
		if _, ok := byPID[s.ParentPID]; ok {
			return ProcDecision{Action: ProcKeepSystem, Reason: "host de console ativo"}
		}
		return ProcDecision{Action: ProcKill, Reason: "conhost órfão"}
	}
	if norm == "antigravity.exe" {
		return classifyAntigravity(s, brokers, openTree, anomalyLimitMB)
	}
	if isLanguageServerName(norm) {
		return classifyLanguageServer(s, brokers, bestHub, children, byPID, anomalyLimitMB)
	}
	if _, ok := standaloneLSPNames[norm]; ok {
		if hasActiveTaskChild(s.PID, children, byPID) {
			return ProcDecision{Action: ProcKeepAndTrim, Reason: "LSP com tarefa ativa"}
		}
		return ProcKillDecision(norm)
	}
	if s.HasOpenWindow || openTree[s.PID] {
		return ProcDecision{Action: ProcKeepAndTrim, Reason: "janela aberta ou árvore ativa"}
	}
	if essentialTree[s.PID] {
		return ProcDecision{Action: ProcKeepAndTrim, Reason: "aplicativo principal protegido"}
	}
	if (s.HasActiveNet || s.IsNetService || netTree[s.PID]) && !isBloatwareName(norm) {
		return ProcDecision{Action: ProcKeepAndTrim, Reason: "conexão de rede ativa preservada"}
	}
	if desktopEnumOK {
		return ProcKillDecision(norm)
	}
	if isBloatwareName(norm) {
		return ProcDecision{Action: ProcKill, Reason: "utilitário ocioso em segundo plano"}
	}
	return ProcDecision{Action: ProcKeepAndTrim, Reason: "preservado (enumeração de desktop indisponível)"}
}

func isBloatwareName(norm string) bool {
	_, ok := bloatwareNames[norm]
	return ok
}

// ProcKillDecision retorna a decisao padrao de encerramento em segundo plano.
func ProcKillDecision(_ string) ProcDecision {
	return ProcDecision{Action: ProcKill, Reason: "segundo plano sem janela aberta"}
}

// ClassifyProcesses decide individualmente o destino de cada processo em execucao.
func ClassifyProcesses(snaps []ProcSnapshot, selfPID uint32, desktopEnumOK bool, anomalyLimitMB int) map[uint32]ProcDecision {
	byPID := make(map[uint32]ProcSnapshot, len(snaps))
	children := make(map[uint32][]uint32, len(snaps))
	brokers := make(map[uint32]bool)
	for _, s := range snaps {
		byPID[s.PID] = s
		if s.ParentPID != 0 && s.ParentPID != s.PID {
			children[s.ParentPID] = append(children[s.ParentPID], s.PID)
		}
		if strings.EqualFold(strings.TrimSpace(s.Name), "antigravity.exe") && !strings.Contains(strings.ToLower(s.CmdLine), "--type=") {
			brokers[s.PID] = true
		}
	}
	selfAncestors := collectAncestors(selfPID, byPID)
	svcTree := buildServiceTree(snaps, children, byPID)
	openTree := buildOpenTree(snaps, byPID, children)
	essentialTree := buildEssentialTree(snaps, children)
	netTree := buildActiveNetTree(snaps, byPID, children)
	bestHub := findActiveLanguageServerHubs(snaps, brokers)
	decisions := make(map[uint32]ProcDecision, len(snaps))
	for _, s := range snaps {
		norm := strings.ToLower(strings.TrimSpace(s.Name))
		decisions[s.PID] = decideSingleProc(s, norm, byPID, children, brokers, selfAncestors, svcTree, openTree, essentialTree, netTree, bestHub, desktopEnumOK, anomalyLimitMB)
	}
	return decisions
}
