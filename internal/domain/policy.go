package domain

import "strings"

// AutoCleanCooldownSec evita ciclos repetidos quando a RAM permanece acima do limiar.
const AutoCleanCooldownSec int64 = 60

var protectedNames = map[string]struct{}{
	"system":               {},
	"idle":                 {},
	"registry":             {},
	"smss.exe":             {},
	"csrss.exe":            {},
	"wininit.exe":          {},
	"services.exe":         {},
	"lsass.exe":            {},
	"svchost.exe":          {},
	"dwm.exe":              {},
	"explorer.exe":         {},
	"spoolsv.exe":          {},
	"audiodg.exe":          {},
	"conhost.exe":          {},
	"fontdrvhost.exe":      {},
	"sihost.exe":           {},
	"taskmgr.exe":          {},
	"ramkiller.exe":        {},
	"language_server.exe":  {},
	"vgc.exe":              {},
	"vgtray.exe":           {},
	"memory compression":   {},
}

var allRegionSpecs = []RegionSpec{
	{Flag: RegionWorkingSet, Label: "Working Set (processos)", FreezeRisk: false, RequiresAdmin: false},
	{Flag: RegionSystemFileCache, Label: "Cache de Arquivos do Sistema", FreezeRisk: false, RequiresAdmin: true},
	{Flag: RegionStandbyPriority0, Label: "Lista de Espera (baixa prioridade)", FreezeRisk: false, RequiresAdmin: true},
	{Flag: RegionStandbyList, Label: "Lista de Espera Completa*", FreezeRisk: true, RequiresAdmin: true},
	{Flag: RegionModifiedPageList, Label: "Lista de Páginas Modificadas*", FreezeRisk: true, RequiresAdmin: true},
	{Flag: RegionCombineMemoryLists, Label: "Combinar Listas de Memória", FreezeRisk: false, RequiresAdmin: true},
	{Flag: RegionRegistryCache, Label: "Cache do Registro", FreezeRisk: false, RequiresAdmin: true},
	{Flag: RegionModifiedFileCache, Label: "Cache de Arquivos Modificados", FreezeRisk: false, RequiresAdmin: true},
}

// AllRegionSpecs retorna todas as regioes suportadas na ordem canônica de exibicao.
func AllRegionSpecs() []RegionSpec {
	out := make([]RegionSpec, len(allRegionSpecs))
	copy(out, allRegionSpecs)
	return out
}

// RegionLabels converte uma mascara em lista legivel de nomes em PT-BR.
func RegionLabels(mask CleanRegion) []string {
	var labels []string
	for _, spec := range allRegionSpecs {
		if mask&spec.Flag == spec.Flag {
			labels = append(labels, spec.Label)
		}
	}
	return labels
}

// EffectiveMask resolve a mascara real aplicada removendo listas de risco no modo automatico.
func EffectiveMask(cfg Config, src CleanSource, overrideMask CleanRegion) CleanRegion {
	mask := overrideMask
	if mask == 0 {
		mask = cfg.CleanMask
	}
	mask &= MaskAll
	if mask == 0 {
		mask = MaskDefault
	}
	if src == SourceAuto && !cfg.AllowStandbyInAuto {
		mask &^= MaskFreezeRisk
		if mask == 0 {
			mask = RegionWorkingSet
		}
	}
	return mask
}

// ShouldAutoClean avalia gatilhos de porcentagem e intervalo com trava de cooldown.
func ShouldAutoClean(cfg Config, physicalPct int, lastCleanUnix int64, nowUnix int64) (bool, CleanSource) {
	if lastCleanUnix > nowUnix {
		return false, ""
	}
	elapsed := nowUnix - lastCleanUnix
	if cfg.AutoCleanThresholdEnabled && physicalPct >= cfg.AutoCleanThresholdPct {
		if lastCleanUnix == 0 || elapsed >= AutoCleanCooldownSec {
			return true, SourceAuto
		}
	}
	if cfg.AutoCleanIntervalEnabled && cfg.AutoCleanIntervalMin > 0 {
		intervalSec := int64(cfg.AutoCleanIntervalMin) * 60
		if lastCleanUnix > 0 && elapsed >= intervalSec && elapsed >= AutoCleanCooldownSec {
			return true, SourceAuto
		}
	}
	return false, ""
}

// IsProtectedProcess impede reciclagem de processos vitais do Windows e foco ativo.
func IsProtectedProcess(name string, pid uint32, selfPID uint32, foregroundPID uint32, protectForeground bool) bool {
	if pid <= 4 {
		return true
	}
	if selfPID != 0 && pid == selfPID {
		return true
	}
	if protectForeground && foregroundPID != 0 && pid == foregroundPID {
		return true
	}
	norm := strings.ToLower(strings.TrimSpace(name))
	if norm == "" {
		return false
	}
	_, exists := protectedNames[norm]
	return exists
}

// IsAnomalyProcess sinaliza subprocesso com consumo privado acima do teto configurado.
func IsAnomalyProcess(privateBytes uint64, limitMB int) bool {
	if limitMB <= 0 {
		limitMB = 2560
	}
	return privateBytes >= uint64(limitMB)*1024*1024
}
