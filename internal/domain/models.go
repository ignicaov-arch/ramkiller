package domain

// CleanRegion representa uma mascara de bits das regioes limpaveis no Windows.
type CleanRegion uint32

const (
	RegionWorkingSet         CleanRegion = 1 << 0
	RegionSystemFileCache    CleanRegion = 1 << 1
	RegionStandbyPriority0   CleanRegion = 1 << 2
	RegionStandbyList        CleanRegion = 1 << 3
	RegionModifiedPageList   CleanRegion = 1 << 4
	RegionCombineMemoryLists CleanRegion = 1 << 5
	RegionRegistryCache      CleanRegion = 1 << 6
	RegionModifiedFileCache  CleanRegion = 1 << 7

	// MaskDefault mantem limpeza segura sem listas de espera completas.
	MaskDefault = RegionWorkingSet | RegionSystemFileCache | RegionStandbyPriority0 |
		RegionCombineMemoryLists | RegionRegistryCache | RegionModifiedFileCache
	MaskFreezeRisk = RegionStandbyList | RegionModifiedPageList
	MaskAll        = MaskDefault | MaskFreezeRisk
)

// CleanSource identifica quem disparou a operacao de limpeza.
type CleanSource string

const (
	SourceManual CleanSource = "manual"
	SourceAuto   CleanSource = "auto"
	SourceHotkey CleanSource = "hotkey"
	SourceCLI    CleanSource = "cli"
)

// Severity representa o nivel semantico do uso de RAM na bandeja e cards.
type Severity string

const (
	SeverityNormal  Severity = "normal"
	SeverityWarning Severity = "warning"
	SeverityDanger  Severity = "danger"
)

// TrayAction define o comportamento de clique no icone da bandeja.
type TrayAction int

const (
	ActionToggleWindow TrayAction = 0
	ActionCleanMemory  TrayAction = 1
	ActionOpenTaskMgr  TrayAction = 2
)

// ProcAction define o destino cirurgico aplicado a cada processo na limpeza.
type ProcAction string

const (
	ProcKeepSystem  ProcAction = "keep_system"
	ProcKeepAndTrim ProcAction = "keep_trim"
	ProcKill        ProcAction = "kill"
)

// ProcSnapshot captura o estado de arvore, sessao, janela e consumo de um processo.
type ProcSnapshot struct {
	PID             uint32 `json:"pid"`
	ParentPID       uint32 `json:"parentPid"`
	SessionID       uint32 `json:"sessionId"`
	Name            string `json:"name"`
	CmdLine         string `json:"cmdLine,omitempty"`
	PrivateBytes    uint64 `json:"privateBytes"`
	WorkingSetBytes uint64 `json:"workingSetBytes"`
	HasOpenWindow   bool   `json:"hasOpenWindow"`
	CPUBusy         bool   `json:"cpuBusy"`
	HasActiveNet    bool   `json:"hasActiveNet,omitempty"`
	IsNetService    bool   `json:"isNetService,omitempty"`
}

// ProcDecision representa a decisao deterministica do classificador para um PID.
type ProcDecision struct {
	Action ProcAction `json:"action"`
	Reason string     `json:"reason"`
}

// BankStat descreve metricas de um subsistema de memoria (Fisica, Paginacao, Cache).
type BankStat struct {
	UsedBytes    uint64  `json:"usedBytes"`
	FreeBytes    uint64  `json:"freeBytes"`
	TotalBytes   uint64  `json:"totalBytes"`
	Percent      int     `json:"percent"`
	PercentFloat float64 `json:"percentFloat"`
}

// ProcessInfo descreve um processo ativo consumindo memoria privada e working set.
type ProcessInfo struct {
	PID             uint32 `json:"pid"`
	Name            string `json:"name"`
	PrivateBytes    uint64 `json:"privateBytes"`
	WorkingSetBytes uint64 `json:"workingSetBytes"`
	Protected       bool   `json:"protected"`
	OpenWindow      bool   `json:"openWindow"`
	Killable        bool   `json:"killable"`
	Anomaly         bool   `json:"anomaly"`
}

// MemoryStats consolida telemetria em tempo real da memoria do sistema.
type MemoryStats struct {
	Physical      BankStat      `json:"physical"`
	Pagefile      BankStat      `json:"pagefile"`
	SystemCache   BankStat      `json:"systemCache"`
	TopProcesses  []ProcessInfo `json:"topProcesses"`
	Elevated      bool          `json:"elevated"`
	TimestampUnix int64         `json:"timestampUnix"`
}

// CleanResult registra o resultado objetivo de uma rodada de limpeza.
type CleanResult struct {
	Source              CleanSource `json:"source"`
	Mask                CleanRegion `json:"mask"`
	BeforeUsedBytes     uint64      `json:"beforeUsedBytes"`
	AfterUsedBytes      uint64      `json:"afterUsedBytes"`
	FreedBytes          uint64      `json:"freedBytes"`
	BeforePagefileBytes uint64      `json:"beforePagefileBytes"`
	AfterPagefileBytes  uint64      `json:"afterPagefileBytes"`
	PagefileFreedBytes  uint64      `json:"pagefileFreedBytes"`
	KilledProcs         int         `json:"killedProcs"`
	KilledNames         []string    `json:"killedNames,omitempty"`
	TrimmedProcs        int         `json:"trimmedProcs"`
	KernelExecuted      bool        `json:"kernelExecuted"`
	Elevated            bool        `json:"elevated"`
	DurationMs          int64       `json:"durationMs"`
	RegionNames         []string    `json:"regionNames"`
	Warnings            []string    `json:"warnings,omitempty"`
}

// RegionSpec descreve metadados de cada regiao de limpeza exibida ao usuario.
type RegionSpec struct {
	Flag          CleanRegion
	Label         string
	FreezeRisk    bool
	RequiresAdmin bool
}
