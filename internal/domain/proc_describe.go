package domain

import "strings"

var friendlyAppLabels = map[string]string{
	"whatsapp.root.exe":           "WhatsApp Desktop",
	"whatsapp.exe":                "WhatsApp Desktop",
	"ayugram.exe":                 "AyuGram · Cliente Telegram",
	"telegram.exe":                "Telegram Desktop",
	"discord.exe":                 "Discord · Voz e Chat",
	"tabtip.exe":                  "Teclado Virtual / Emojis (TIP)",
	"searchhost.exe":              "Busca Web do Menu Iniciar",
	"widgets.exe":                 "Widgets da Barra de Tarefas",
	"widgetservice.exe":           "Serviço de Widgets Windows",
	"msmpeng.exe":                 "Windows Defender Antivírus",
	"mpdefendercoreservice.exe":   "Núcleo Windows Defender",
	"mpdefendercorereceiver.exe":  "Monitor Windows Defender",
	"nissrv.exe":                  "Inspeção de Rede Defender",
	"explorer.exe":                "Windows Explorer · Shell",
	"dwm.exe":                     "Gerenciador de Janelas DWM",
	"svchost.exe":                 "Host de Serviços do Windows",
	"runtimebroker.exe":           "Gerenciador de Permissões UWP",
	"shellexperiencehost.exe":     "Menu Iniciar e Notificações",
	"startmenuexperiencehost.exe": "Experiência do Menu Iniciar",
	"textinputhost.exe":           "Painel de Entrada de Texto",
	"ctfmon.exe":                  "Monitor de Teclado e Idioma",
	"chrome.exe":                  "Google Chrome",
	"brave.exe":                   "Navegador Brave",
	"msedge.exe":                  "Microsoft Edge",
	"teamviewer.exe":              "TeamViewer · Acesso Remoto",
	"anydesk.exe":                 "AnyDesk · Acesso Remoto",
	"nvcontainer.exe":             "Serviço de Driver NVIDIA",
	"nvidia overlay.exe":          "Overlay / Gravação NVIDIA",
	"rtkauduservice64.exe":        "Driver de Áudio Realtek",
	"audiodg.exe":                 "Motor de Áudio do Windows",
	"vgtray.exe":                  "Riot Vanguard · Bandeja",
	"node.exe":                    "Runtime Node.js",
	"ramkiller.exe":               "RamKiller · Monitor de RAM",
	"smartscreen.exe":             "Filtro Windows SmartScreen",
	"predatorsense.exe":           "Acer PredatorSense",
}

// ExtractWebViewHost extrai o executavel hospedeiro de --webview-exe-name=<exe>.
func ExtractWebViewHost(cmdLine string) string {
	lower := strings.ToLower(cmdLine)
	const key = "--webview-exe-name="
	idx := strings.Index(lower, key)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(cmdLine[idx+len(key):])
	if rest == "" {
		return ""
	}
	if rest[0] == '"' {
		if end := strings.IndexByte(rest[1:], '"'); end >= 0 {
			return strings.TrimSpace(rest[1 : 1+end])
		}
	}
	for i := 0; i < len(rest); i++ {
		if rest[i] == ' ' || rest[i] == '\t' {
			return strings.TrimSpace(rest[:i])
		}
	}
	return rest
}

// IsBloatwareWebView identifica WebView2 acoplado a bloatware do sistema (Busca Web/Widgets).
func IsBloatwareWebView(cmdLine string) bool {
	host := strings.ToLower(strings.TrimSpace(ExtractWebViewHost(cmdLine)))
	if host == "" {
		return false
	}
	_, isBloat := bloatwareNames[host]
	return isBloat
}

func hasBloatwareAncestor(pid uint32, byPID map[uint32]ProcSnapshot, openTree map[uint32]bool) bool {
	cur := pid
	for i := 0; i < 32 && cur > 0; i++ {
		snap, ok := byPID[cur]
		if !ok || snap.SessionID == 0 {
			break
		}
		norm := strings.ToLower(strings.TrimSpace(snap.Name))
		if _, isBloat := bloatwareNames[norm]; isBloat && !openTree[cur] {
			return true
		}
		if IsBloatwareWebView(snap.CmdLine) && !openTree[cur] {
			return true
		}
		if snap.ParentPID == cur {
			break
		}
		cur = snap.ParentPID
	}
	return false
}

func chromiumRoleLabel(cmdLower string) string {
	switch {
	case strings.Contains(cmdLower, "network.mojom."):
		return "Serviço de Rede"
	case strings.Contains(cmdLower, "storage.mojom."):
		return "Armazenamento"
	case strings.Contains(cmdLower, "audio.mojom."):
		return "Serviço de Áudio"
	case strings.Contains(cmdLower, "--type=gpu-process"):
		return "Aceleração GPU"
	case strings.Contains(cmdLower, "--type=renderer"):
		return "Interface / Renderização"
	case strings.Contains(cmdLower, "--type=crashpad-handler"):
		return "Relatório de Falhas"
	case strings.Contains(cmdLower, "--embedded-browser-webview"):
		return "Motor Principal WebView2"
	default:
		return "Subprocesso"
	}
}

func resolveHostDisplay(host string) string {
	norm := strings.ToLower(strings.TrimSpace(host))
	switch norm {
	case "whatsapp.root.exe", "whatsapp.exe":
		return "WhatsApp"
	case "searchhost.exe":
		return "Busca Web Windows"
	case "widgets.exe", "widgetservice.exe":
		return "Widgets Windows"
	case "teamviewer.exe":
		return "TeamViewer"
	case "anydesk.exe":
		return "AnyDesk"
	}
	return strings.TrimSuffix(strings.TrimSuffix(host, ".exe"), ".EXE")
}

// DescribeProcess retorna um subtitulo curto e claro explicando o papel real do processo na RAM.
func DescribeProcess(s ProcSnapshot, byPID map[uint32]ProcSnapshot, dec ProcDecision) string {
	norm := strings.ToLower(strings.TrimSpace(s.Name))
	cmdLower := strings.ToLower(s.CmdLine)

	if norm == "msedgewebview2.exe" {
		host := ExtractWebViewHost(s.CmdLine)
		if host == "" {
			if p, ok := byPID[s.ParentPID]; ok && !strings.EqualFold(p.Name, "msedgewebview2.exe") {
				host = p.Name
			}
		}
		role := chromiumRoleLabel(cmdLower)
		if host != "" {
			return resolveHostDisplay(host) + " · " + role
		}
		return "WebView2 · " + role
	}

	if isLanguageServerName(norm) {
		return "Antigravity · Motor de IA e Código"
	}
	if norm == "antigravity.exe" {
		if strings.Contains(cmdLower, "--type=") {
			return "Antigravity IDE · " + chromiumRoleLabel(cmdLower)
		}
		return "Antigravity IDE · Processo Principal"
	}
	if lbl, ok := friendlyAppLabels[norm]; ok {
		if strings.Contains(cmdLower, "--type=") && (norm == "chrome.exe" || norm == "brave.exe" || norm == "msedge.exe") {
			return lbl + " · " + chromiumRoleLabel(cmdLower)
		}
		return lbl
	}
	if dec.Action == ProcKeepSystem {
		return "Componente / Serviço do Sistema"
	}
	if dec.Reason == "conexão de rede ativa preservada" {
		return "Processo com conexão de rede ativa"
	}
	if dec.Action == ProcKill {
		return "Segundo plano sem janela aberta"
	}
	return "Aplicativo ativo do usuário"
}
