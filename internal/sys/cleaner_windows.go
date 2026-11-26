//go:build windows

package sys

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

// ErrCleanInProgress indica que uma limpeza ja esta em andamento.
var ErrCleanInProgress = errors.New("limpeza ja esta em andamento")

// Engine coordena limpeza concorrente-segura entre User-Space e NT Kernel.
type Engine struct {
	cleaning atomic.Bool
}

// NewEngine cria uma instancia do motor duplo de limpeza.
func NewEngine() *Engine {
	return &Engine{}
}

// IsCleaning informa se ha limpeza em execucao no momento.
func (e *Engine) IsCleaning() bool {
	return e.cleaning.Load()
}

// Clean executa encerramento de processos ociosos, reciclagem de Working Set e limpeza NT Kernel.
func (e *Engine) Clean(
	ctx context.Context,
	src domain.CleanSource,
	cfg domain.Config,
	overrideMask domain.CleanRegion,
) (domain.CleanResult, error) {
	if !e.cleaning.CompareAndSwap(false, true) {
		return domain.CleanResult{}, ErrCleanInProgress
	}
	defer e.cleaning.Store(false)

	if err := ctx.Err(); err != nil {
		return domain.CleanResult{}, err
	}

	start := time.Now()
	mask := domain.EffectiveMask(cfg, src, overrideMask)
	beforeStats, err := ReadMemoryStats(0, cfg.AnomalyLimitMB)
	if err != nil {
		return domain.CleanResult{}, err
	}

	elevated := IsElevated()
	if elevated {
		_ = EnableMemoryPrivileges()
	}

	res := domain.CleanResult{
		Source:              src,
		Mask:                mask,
		BeforeUsedBytes:     beforeStats.Physical.UsedBytes,
		BeforePagefileBytes: beforeStats.Pagefile.UsedBytes,
		Elevated:            elevated,
		RegionNames:         domain.RegionLabels(mask),
	}

	if mask&domain.RegionWorkingSet != 0 {
		allowKill := cfg.KillBackgroundApps
		killed, names, trimmed := ExecuteProcessPlan(cfg, allowKill)
		res.KilledProcs = killed
		res.KilledNames = names
		res.TrimmedProcs = trimmed
	}

	if err := ctx.Err(); err != nil {
		return res, err
	}

	if res.Elevated {
		kernelOK, warns := RunKernelClean(mask)
		res.KernelExecuted = kernelOK
		res.Warnings = append(res.Warnings, warns...)
	}

	runtime.GC()
	TrimSelf()

	afterStats, err := ReadMemoryStats(0, cfg.AnomalyLimitMB)
	if err == nil {
		res.AfterUsedBytes = afterStats.Physical.UsedBytes
		if res.BeforeUsedBytes > res.AfterUsedBytes {
			res.FreedBytes = res.BeforeUsedBytes - res.AfterUsedBytes
		}
		res.AfterPagefileBytes = afterStats.Pagefile.UsedBytes
		if res.BeforePagefileBytes > res.AfterPagefileBytes {
			res.PagefileFreedBytes = res.BeforePagefileBytes - res.AfterPagefileBytes
		}
	} else {
		res.AfterUsedBytes = res.BeforeUsedBytes
		res.AfterPagefileBytes = res.BeforePagefileBytes
	}
	res.DurationMs = time.Since(start).Milliseconds()

	if cfg.LogResults {
		_ = AppendCleanLog(res)
	}
	return res, nil
}

// CleanPagefile libera consumo real no arquivo de paginacao sem empurrar RAM aberta para o disco.
func (e *Engine) CleanPagefile(ctx context.Context, cfg domain.Config) (domain.CleanResult, error) {
	if !e.cleaning.CompareAndSwap(false, true) {
		return domain.CleanResult{}, ErrCleanInProgress
	}
	defer e.cleaning.Store(false)

	if err := ctx.Err(); err != nil {
		return domain.CleanResult{}, err
	}

	start := time.Now()
	beforeStats, err := ReadMemoryStats(0, cfg.AnomalyLimitMB)
	if err != nil {
		return domain.CleanResult{}, err
	}

	elevated := IsElevated()
	if elevated {
		_ = EnableMemoryPrivileges()
	}

	res := domain.CleanResult{
		Source:              domain.SourcePagefile,
		Mask:                domain.RegionCombineMemoryLists | domain.RegionRegistryCache | domain.RegionModifiedFileCache | domain.RegionSystemFileCache | domain.RegionStandbyPriority0,
		BeforeUsedBytes:     beforeStats.Physical.UsedBytes,
		BeforePagefileBytes: beforeStats.Pagefile.UsedBytes,
		Elevated:            elevated,
		RegionNames:         []string{"Paginação / Commit Real"},
	}

	killed, names := ExecutePagefileProcessPlan(cfg)
	res.KilledProcs = killed
	res.KilledNames = names

	if res.Elevated {
		kernelOK, warns := RunPagefileKernelClean()
		res.KernelExecuted = kernelOK
		res.Warnings = append(res.Warnings, warns...)
	}

	runtime.GC()
	TrimSelf()

	afterStats, err := ReadMemoryStats(0, cfg.AnomalyLimitMB)
	if err == nil {
		res.AfterUsedBytes = afterStats.Physical.UsedBytes
		if res.BeforeUsedBytes > res.AfterUsedBytes {
			res.FreedBytes = res.BeforeUsedBytes - res.AfterUsedBytes
		}
		res.AfterPagefileBytes = afterStats.Pagefile.UsedBytes
		if res.BeforePagefileBytes > res.AfterPagefileBytes {
			res.PagefileFreedBytes = res.BeforePagefileBytes - res.AfterPagefileBytes
		}
	} else {
		res.AfterUsedBytes = res.BeforeUsedBytes
		res.AfterPagefileBytes = res.BeforePagefileBytes
	}
	res.DurationMs = time.Since(start).Milliseconds()

	if cfg.LogResults {
		_ = AppendCleanLog(res)
	}
	return res, nil
}
