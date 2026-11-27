package domain

import "strings"

// AutoCleanCooldownSec evita ciclos repetidos quando a RAM permanece acima do limiar.
const AutoCleanCooldownSec int64 = 60

var protectedNames = map[string]struct{}{
	"system":                    {},
	"idle":                      {},
	"registry":                  {},
	"secure system":             {},
	"memory compression":        {},
	"smss.exe":                  {},
	"csrss.exe":                 {},
	"wininit.exe":               {},
	"winlogon.exe":              {},
	"services.exe":              {},
	"lsass.exe":                 {},
	"lsaiso.exe":                {},
	"ngciso.exe":                {},
	"svchost.exe":               {},
	"dwm.exe":                   {},
	"explorer.exe":              {},
	"sihost.exe":                {},
	"fontdrvhost.exe":           {},
	"ctfmon.exe":                {},
	"taskhostw.exe":             {},
	"runtimebroker.exe":         {},
	"shellexperiencehost.exe":   {},
	"startmenuexperiencehost.exe": {},
	"textinputhost.exe":         {},
	"dllhost.exe":               {},
	"spoolsv.exe":               {},
	"audiodg.exe":                 {},
	"rtkauduservice64.exe":        {},
	"searchindexer.exe":           {},
	"taskmgr.exe":                 {},
	"ramkiller.exe":               {},
	"msmpeng.exe":                 {},
	"mpdefendercoreservice.exe":   {},
	"mpdefendercorereceiver.exe":  {},
	"defendersessionhelper.exe":   {},
	"nissrv.exe":                {},
	"securityhealthsystray.exe": {},
	"securityhealthservice.exe": {},
	"vgc.exe":                   {},
	"vgtray.exe":                {},
	"nvdisplay.container.exe":   {},
	"nvcontainer.exe":           {},
	"nvidia overlay.exe":        {},
	"nvidia share.exe":          {},
	"anydesk.exe":               {},
	"teamviewer.exe":            {},
	"teamviewer_service.exe":    {},
	"teamviewer_desktop.exe":    {},
	"tv_w32.exe":                {},
	"tv_x64.exe":                {},
	"parsecd.exe":               {},
	"pservice.exe":              {},
	"rustdesk.exe":              {},
}

var allRegionSpecs = []RegionSpec{
	{Flag: RegionWorkingSet, Label: "Working Set (vista+)", FreezeRisk: false, RequiresAdmin: false},
	{Flag: RegionSystemFileCache, Label: "Cache do Sistema", FreezeRisk: false, RequiresAdmin: true},
	{Flag: RegionStandbyPriority0, Label: "Espera s/ prior. (vista+)", FreezeRisk: false, RequiresAdmin: true},
	{Flag: RegionStandbyList, Label: "Lista de Espera* (vista+)", FreezeRisk: true, RequiresAdmin: true},
	{Flag: RegionModifiedPageList, Label: "Pág. Modificada* (vista+)", FreezeRisk: true, RequiresAdmin: true},
	{Flag: RegionCombineMemoryLists, Label: "Combinar listas (win10+)", FreezeRisk: false, RequiresAdmin: true},
	{Flag: RegionRegistryCache, Label: "Cache do Registro", FreezeRisk: false, RequiresAdmin: true},
	{Flag: RegionModifiedFileCache, Label: "Arquivos Modificados", FreezeRisk: false, RequiresAdmin: true},
}

// AllRegionSpecs retorna todas as regioes suportadas na ordem canonica de exibicao.
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

// EffectiveMask resolve a mascara unificada aplicada em limpeza manual, automatica, CLI e atalho.
func EffectiveMask(cfg Config, _ CleanSource, overrideMask CleanRegion) CleanRegion {
	mask := overrideMask
	if mask == 0 {
		mask = cfg.CleanMask
	}
	mask &= MaskAll
	if mask == 0 {
		mask = MaskAll
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

// IsProtectedProcess impede encerramento de processos vitais do Windows e foco ativo.
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
