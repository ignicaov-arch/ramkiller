# RamKiller

Utilitário nativo para Windows 10 e Windows 11 focado na barra de tarefas (System Tray) para limpeza real de memória RAM, eliminação cirúrgica de processos ociosos em segundo plano, autoregeneração automática e controle de kernel NT sem telemetria ou recursos supérfluos.

## O que o RamKiller v2.0 faz

1. **Todas as Clean Areas Ativas por Padrão (`MaskAll` + Modo Kernel):**
   - Habilita `SeProfileSingleProcessPrivilege`, `SeIncreaseQuotaPrivilege` e `SeDebugPrivilege` e executa as **8 regiões reais do kernel NT** via `ntdll.dll!NtSetSystemInformation` e `SetSystemFileCacheSize`:
     - Cache Working Set (vista+)
     - Cache do Sistema
     - Lista de Espera s/ prioridade (vista+)
     - Lista de Espera* (vista+)
     - Lista de Paginação Modificada* (vista+)
     - Combinar Listas de Memória (win10+)
     - Cache do Registro e Cache de Arquivos Modificados

2. **Preserva Abertos e Principais, Encerra o Resto (`WinSta0\Default`):**
   - Enumera todas as janelas abertas reais em `WinSta0\Default` (`EnumDesktopWindows` + `DwmGetWindowAttribute` + `EnumChildWindows`), protegendo integralmente as árvores dos aplicativos abertos pelo usuário, serviços do sistema (`services.exe`, drivers, áudio Realtek, Vanguard, Defender) e aplicativos principais.
   - **Antigravity + `language_server`:** preserva o `Antigravity.exe` principal e o hub ativo do `language_server` (aplicando reciclagem de Working Set sem interromper tarefas ou IDE), enquanto encerra instâncias órfãs/duplicadas de `language_server`, LSPs ociosos e renderers vazadores acima do teto configurado.
   - Encerra (`TerminateProcess`) o restante dos processos descartáveis em segundo plano sem janela aberta (bloatware, WebView2 órfãos, updaters e helpers ociosos).

3. **Autoregeneração (`--guardian`), Inicialização no Boot e Pausa Temporária:**
   - Registra inicialização automática em `HKCU\...\Run` e na tarefa agendada elevada `\RamKiller_AutoStart`.
   - Mantém um processo companheiro (`--guardian`) que regenera o `RamKiller` automaticamente em ~2,5s caso seja encerrado externamente.
   - Inclui botão **Encerrar até Reiniciar o PC** (no painel e no menu da bandeja) que grava o `BootID` atual do kernel e interrompe o guardião exclusivamente até o próximo reinício do computador.

## Uso por linha de comando (CLI)

```powershell
ramkiller.exe                     # Inicia elevado com painel principal e bandeja
ramkiller.exe --minimized         # Inicia minimizado na bandeja e ativa o guardião
ramkiller.exe --status            # Exibe telemetria atual e Top 5 processos
ramkiller.exe --audit-procs       # Mostra a classificação cirúrgica de cada processo
ramkiller.exe --clean             # 1 comando: limpa RAM, Paginação (pagefile.sys), 2º plano e 8 regiões Kernel
ramkiller.exe --install-shortcut  # Cria/atualiza atalho no Menu Iniciar e inicialização no Windows
ramkiller.exe --pause-until-reboot # Encerra painel e guardião até reiniciar o PC
ramkiller.exe --json              # Formata saída CLI em JSON
```

## Compilação e testes locais

Requisitos: Go 1.26+ e Node.js 20+ (zero dependências externas no `go.mod`).

```powershell
npm run lint   # Valida limite modular (<= 200 linhas) e go vet ./...
npm test       # Executa suíte de testes unitários, kernel, UI e processos
npm run build  # Compila dist/ramkiller.exe (-H=windowsgui -s -w, CGO_ENABLED=0)
```

## Licença

MIT. Consulte [LICENSE](LICENSE).
