# RamKiller

Utilitário nativo para Windows 10 e Windows 11 focado na barra de tarefas (System Tray) para monitoramento em tempo real e recuperação cirúrgica de memória RAM, sem telemetria, sem WebView, sem instaladores pesados e sem recursos supérfluos.

## Por que o RamKiller existe

A maioria dos otimizadores de memória comete um de dois erros:
1. Trava ou fecha aplicativos do usuário sem necessidade, ou depende exclusivamente de chamadas globais cegas que exigem UAC para qualquer ação.
2. Enche a bandeja do Windows com telas legadas em Win32 cinza, traduções misturadas e menus de atualização ou doação.

O **RamKiller** combina duas camadas reais em um único executável Go puro (~2,8 MB, sem CGO, ~3 a 7 MB de RAM em execução):

1. **Camada 1 (Modo Usuário, funciona sem Admin):**
   - Recicla páginas ociosas de processos acessíveis com `K32EmptyWorkingSet` em menos de 50 ms.
   - Mantém escudo ativo para processos vitais do Windows (`csrss.exe`, `dwm.exe`, `explorer.exe`, `lsass.exe`), `language_server.exe`, Vanguard (`vgc.exe`, `vgtray.exe`), o próprio `ramkiller.exe` e a janela em foco durante limpezas automáticas.
   - Permite reciclar individualmente qualquer processo listado no Top 5 do painel sem encerrar nenhuma janela.

2. **Camada 2 (Modo Kernel, quando elevado como Administrador):**
   - Habilita `SeProfileSingleProcessPrivilege`, `SeIncreaseQuotaPrivilege` e `SeDebugPrivilege`.
   - Executa seletivamente as 8 regiões do kernel NT via `ntdll.dll!NtSetSystemInformation` e `SetSystemFileCacheSize`:
     - Working Set (`MemoryEmptyWorkingSets`)
     - Cache de Arquivos do Sistema (`SetSystemFileCacheSize` / classe `81`)
     - Lista de Espera de Baixa Prioridade (`MemoryPurgeLowPriorityStandbyList`)
     - Combinar Listas de Memória (`SystemCombinePhysicalMemoryInformation`, classe `130`)
     - Cache do Registro (`SystemRegistryReconciliationInformation`, classe `155`)
     - Cache de Arquivos Modificados (`FlushFileBuffers` em volumes fixos)
     - Opcional manual: Lista de Espera Completa (`MemoryPurgeStandbyList`) e Lista de Páginas Modificadas (`MemoryFlushModifiedList`), bloqueadas por padrão no disparo automático para evitar micro-travamentos.

## Recursos principais

- **Ícone dinâmico na bandeja:** exibe a porcentagem de uso físico da RAM com suavização ClearType, cantos arredondados e cores semânticas configuráveis (Normal, Alerta e Perigo).
- **Tooltip detalhado:** mostra Física, Arquivo de Paginação e Cache do Sistema ao passar o mouse.
- **Menu de contexto 100% em Português do Brasil:** limpar memória, limpeza profunda, seleção de áreas, gatilho automático por porcentagem ou intervalo em minutos e configurações.
- **Painel Dark Pro Windows 11:** 3 cartões de telemetria, Top 5 processos por consumo privado (com destaque acima do teto configurável de 2,5 GB) e ação individual `Reciclar`.
- **Atalho global e inicialização:** suporte a `RegisterHotKey` (`Ctrl + F1`, `F2`, `F3` ou `F9`), registro em `HKCU\...\Run` e tarefa agendada opcional para iniciar elevado sem aviso UAC.

## Uso por linha de comando (CLI)

```powershell
ramkiller.exe              # Abre o painel principal e fixa o ícone na bandeja
ramkiller.exe --minimized  # Inicia minimizado diretamente na bandeja
ramkiller.exe --status     # Mostra telemetria atual e Top 5 processos
ramkiller.exe --clean      # Executa limpeza segura com a máscara configurada
ramkiller.exe --clean-full # Executa limpeza profunda em todas as regiões
ramkiller.exe --json       # Formata saída de --status ou --clean em JSON
```

## Compilação e testes locais

Requisitos: Go 1.26+ e Node.js 20+ (sem dependências externas no `go.mod`).

```powershell
npm run lint   # Valida limite modular (< 200 linhas) e go vet ./...
npm test       # Executa suíte de testes unitários, kernel e renderização
npm run build  # Compila dist/ramkiller.exe (-H=windowsgui -s -w, CGO_ENABLED=0)
```

## Licença

MIT. Consulte [LICENSE](LICENSE).
