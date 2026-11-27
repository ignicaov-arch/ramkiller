package domain

import "strings"

var serviceHostContainers = map[string]struct{}{
	"svchost.exe":  {},
	"dllhost.exe":  {},
	"explorer.exe": {},
}

var protectedAppRoots = map[string]struct{}{
	"anydesk.exe":           {},
	"teamviewer.exe":        {},
	"teamviewer_desktop.exe": {},
	"rustdesk.exe":          {},
	"parsecd.exe":           {},
	"tailscale-ipn.exe":     {},
	"cloudflare warp.exe":   {},
	"nordvpn.exe":           {},
}

func propagateChildren(tree map[uint32]bool, queue []uint32, children map[uint32][]uint32) {
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		for _, ch := range children[pid] {
			if !tree[ch] {
				tree[ch] = true
				queue = append(queue, ch)
			}
		}
	}
}

func collectAncestors(pid uint32, byPID map[uint32]ProcSnapshot) map[uint32]bool {
	out := make(map[uint32]bool)
	cur := pid
	for i := 0; i < 64 && cur > 0; i++ {
		if out[cur] {
			break
		}
		out[cur] = true
		snap, ok := byPID[cur]
		if !ok || snap.ParentPID == cur {
			break
		}
		cur = snap.ParentPID
	}
	return out
}

func markUserAncestors(pid uint32, tree map[uint32]bool, queue *[]uint32, byPID map[uint32]ProcSnapshot) {
	cur := pid
	for i := 0; i < 64 && cur > 0; i++ {
		snap, ok := byPID[cur]
		if !ok || (snap.SessionID == 0 && cur != pid) || tree[cur] {
			break
		}
		// Nunca sobe para containers do SO (explorer.exe, svchost.exe, sihost.exe) para nao herdar apps fechados.
		if cur != pid {
			if _, isSys := protectedNames[strings.ToLower(strings.TrimSpace(snap.Name))]; isSys {
				break
			}
		}
		tree[cur] = true
		*queue = append(*queue, cur)
		if snap.ParentPID == cur {
			break
		}
		cur = snap.ParentPID
	}
}

func buildOpenTree(snaps []ProcSnapshot, byPID map[uint32]ProcSnapshot, children map[uint32][]uint32) map[uint32]bool {
	openTree := make(map[uint32]bool)
	var queue []uint32
	for _, s := range snaps {
		if !s.HasOpenWindow {
			continue
		}
		markUserAncestors(s.PID, openTree, &queue, byPID)
	}
	propagateChildren(openTree, queue, children)
	return openTree
}

func buildServiceTree(snaps []ProcSnapshot, children map[uint32][]uint32, byPID map[uint32]ProcSnapshot) map[uint32]bool {
	svc := make(map[uint32]bool)
	var queue []uint32
	for _, s := range snaps {
		n := strings.ToLower(strings.TrimSpace(s.Name))
		if s.SessionID == 0 || s.PID <= 4 {
			svc[s.PID] = true
		}
		// Servicos dedicados em Sessao 0 (e services.exe/wininit.exe) propagam protecao para helpers filhos;
		// containers genericos (svchost/dllhost/explorer) nao propagam para nao blindar bloatware/UWP.
		_, isContainer := serviceHostContainers[n]
		if n == "services.exe" || n == "wininit.exe" || (s.SessionID == 0 && s.PID > 4 && !isContainer) {
			svc[s.PID] = true
			queue = append(queue, s.PID)
		}
	}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		for _, ch := range children[pid] {
			if snap, ok := byPID[ch]; ok {
				if _, isContainer := serviceHostContainers[strings.ToLower(strings.TrimSpace(snap.Name))]; isContainer {
					continue
				}
			}
			if !svc[ch] {
				svc[ch] = true
				queue = append(queue, ch)
			}
		}
	}
	return svc
}

func buildEssentialTree(snaps []ProcSnapshot, children map[uint32][]uint32) map[uint32]bool {
	ess := make(map[uint32]bool)
	var queue []uint32
	for _, s := range snaps {
		n := strings.ToLower(strings.TrimSpace(s.Name))
		_, isEssential := essentialAppNames[n]
		_, isRoot := protectedAppRoots[n]
		if isEssential || isRoot {
			ess[s.PID] = true
			queue = append(queue, s.PID)
		}
	}
	propagateChildren(ess, queue, children)
	return ess
}

func buildActiveNetTree(snaps []ProcSnapshot, byPID map[uint32]ProcSnapshot, children map[uint32][]uint32, openTree map[uint32]bool) map[uint32]bool {
	netTree := make(map[uint32]bool)
	var queue []uint32
	for _, s := range snaps {
		if !s.HasActiveNet && !s.IsNetService {
			continue
		}
		if hasBloatwareAncestor(s.PID, byPID, openTree) {
			continue
		}
		markUserAncestors(s.PID, netTree, &queue, byPID)
	}
	propagateChildren(netTree, queue, children)
	return netTree
}

func isLanguageServerName(norm string) bool {
	return strings.HasPrefix(norm, "language_server") && strings.HasSuffix(norm, ".exe")
}

func hasActiveTaskChild(pid uint32, children map[uint32][]uint32, byPID map[uint32]ProcSnapshot) bool {
	for _, chPID := range children[pid] {
		if ch, ok := byPID[chPID]; ok {
			if _, busy := activeTaskChildNames[strings.ToLower(strings.TrimSpace(ch.Name))]; busy {
				return true
			}
		}
	}
	return false
}

func findActiveLanguageServerHubs(snaps []ProcSnapshot, brokers map[uint32]bool) map[uint32]uint32 {
	bestByParent := make(map[uint32]uint32)
	for _, s := range snaps {
		norm := strings.ToLower(strings.TrimSpace(s.Name))
		if !isLanguageServerName(norm) || !brokers[s.ParentPID] {
			continue
		}
		cmd := strings.ToLower(s.CmdLine)
		if cmd != "" && !strings.Contains(cmd, "--subclient_type hub") {
			continue
		}
		if s.PID > bestByParent[s.ParentPID] {
			bestByParent[s.ParentPID] = s.PID
		}
	}
	return bestByParent
}
