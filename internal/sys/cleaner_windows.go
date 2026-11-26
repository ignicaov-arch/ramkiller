//go:build windows

package sys

import (
	"context"
	"errors"
	"os"
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

// Clean executa limpeza segura com trava atomica e mede o ganho real em bytes.
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

	res := domain.CleanResult{
		Source:          src,
		Mask:            mask,
		BeforeUsedBytes: beforeStats.Physical.UsedBytes,
		Elevated:        IsElevated(),
		RegionNames:     domain.RegionLabels(mask),
	}

	if mask&domain.RegionWorkingSet != 0 {
		protectFG := src == domain.SourceAuto
		trimmed, _ := TrimAllSafe(uint32(os.Getpid()), protectFG)
		res.TrimmedProcs = trimmed
	}

	if err := ctx.Err(); err != nil {
		return res, err
	}

	if res.Elevated {
		_ = EnableMemoryPrivileges()
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
	} else {
		res.AfterUsedBytes = res.BeforeUsedBytes
	}
	res.DurationMs = time.Since(start).Milliseconds()

	if cfg.LogResults {
		_ = AppendCleanLog(res)
	}
	return res, nil
}
