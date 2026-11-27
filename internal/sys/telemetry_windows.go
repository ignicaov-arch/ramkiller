//go:build windows

package sys

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"time"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const (
	sysPageFileInfoClass     = 18
	systemFileCacheInfoClass = 21
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type performanceInfo struct {
	Cb                uint32
	CommitTotal       uintptr
	CommitLimit       uintptr
	CommitPeak        uintptr
	PhysicalTotal     uintptr
	PhysicalAvailable uintptr
	SystemCache       uintptr
	KernelTotal       uintptr
	KernelPaged       uintptr
	KernelNonpaged    uintptr
	PageSize          uintptr
	HandleCount       uint32
	ProcessCount      uint32
	ThreadCount       uint32
}

type systemFileCacheInfo struct {
	CurrentSize                           uintptr
	PeakSize                              uintptr
	PageFaultCount                        uint32
	_                                     uint32
	MinimumWorkingSet                     uintptr
	MaximumWorkingSet                     uintptr
	CurrentSizeIncludingTransitionInPages uintptr
	PeakSizeIncludingTransitionInPages    uintptr
	TransitionRePurposeCount              uint32
	Flags                                 uint32
}

func calcBank(used, total uint64) domain.BankStat {
	if total == 0 {
		return domain.BankStat{}
	}
	if used > total {
		used = total
	}
	free := total - used
	pctFloat := (float64(used) / float64(total)) * 100.0
	pct := int(math.Round(pctFloat))
	if pct > 100 {
		pct = 100
	}
	return domain.BankStat{
		UsedBytes:    used,
		FreeBytes:    free,
		TotalBytes:   total,
		Percent:      pct,
		PercentFloat: pctFloat,
	}
}

func readPagefileInfoNT(pageSize uint64) (usedBytes, totalBytes uint64, ok bool) {
	if pageSize == 0 {
		pageSize = 4096
	}
	buf := make([]byte, 4096)
	var retLen uint32
	st, _, _ := procNtQuerySystemInformation.Call(
		sysPageFileInfoClass,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&retLen)),
	)
	if uint32(st) == statusInfoLenMismatch && retLen > 0 && retLen <= 262144 {
		buf = make([]byte, retLen)
		st, _, _ = procNtQuerySystemInformation.Call(
			sysPageFileInfoClass,
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(len(buf)),
			uintptr(unsafe.Pointer(&retLen)),
		)
	}
	if uint32(st) != 0 || retLen < 16 {
		return 0, 0, false
	}
	limit := int(retLen)
	if limit > len(buf) {
		limit = len(buf)
	}
	for off := 0; off+16 <= limit; {
		nextOff := int(binary.LittleEndian.Uint32(buf[off:]))
		totalPages := uint64(binary.LittleEndian.Uint32(buf[off+4:]))
		usedPages := uint64(binary.LittleEndian.Uint32(buf[off+8:]))
		totalBytes += totalPages * pageSize
		usedBytes += usedPages * pageSize
		if nextOff <= 0 || off+nextOff >= limit {
			break
		}
		off += nextOff
	}
	return usedBytes, totalBytes, totalBytes > 0
}

// ReadMemoryStats coleta telemetria real de RAM Fisica, Paginacao, Cache e Top Processos.
func ReadMemoryStats(topN int, anomalyLimitMB int) (domain.MemoryStats, error) {
	var ms memoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms))
	r, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
	if r == 0 {
		return domain.MemoryStats{}, fmt.Errorf("GlobalMemoryStatusEx falhou: %w", err)
	}

	var stats domain.MemoryStats
	physUsed := uint64(0)
	if ms.TotalPhys > ms.AvailPhys {
		physUsed = ms.TotalPhys - ms.AvailPhys
	}
	stats.Physical = calcBank(physUsed, ms.TotalPhys)

	var pi performanceInfo
	pi.Cb = uint32(unsafe.Sizeof(pi))
	okPerf, _, _ := procK32GetPerformanceInfo.Call(uintptr(unsafe.Pointer(&pi)), uintptr(pi.Cb))
	if okPerf == 0 {
		okPerf, _, _ = procPsapiGetPerformanceInfo.Call(uintptr(unsafe.Pointer(&pi)), uintptr(pi.Cb))
	}
	pageSize := uint64(4096)
	if okPerf != 0 && pi.PageSize > 0 {
		pageSize = uint64(pi.PageSize)
	}
	if pfUsed, pfTotal, pfOK := readPagefileInfoNT(pageSize); pfOK {
		stats.Pagefile = calcBank(pfUsed, pfTotal)
	} else if okPerf != 0 {
		stats.Pagefile = calcBank(uint64(pi.CommitTotal)*pageSize, uint64(pi.CommitLimit)*pageSize)
	}

	var sfci systemFileCacheInfo
	var retLen uint32
	ntStatus, _, _ := procNtQuerySystemInformation.Call(
		systemFileCacheInfoClass,
		uintptr(unsafe.Pointer(&sfci)),
		unsafe.Sizeof(sfci),
		uintptr(unsafe.Pointer(&retLen)),
	)
	if ntStatus == 0 && sfci.PeakSize > 0 {
		cacheTotal := uint64(sfci.PeakSize)
		if cacheTotal < uint64(sfci.CurrentSize) {
			cacheTotal = ms.TotalPhys
		}
		stats.SystemCache = calcBank(uint64(sfci.CurrentSize), cacheTotal)
	} else if okPerf != 0 {
		cacheUsed := uint64(pi.SystemCache) * pageSize
		stats.SystemCache = calcBank(cacheUsed, ms.TotalPhys)
	}

	if topN != 0 {
		listLimit := topN
		if topN < 0 {
			listLimit = 0
		}
		procs, _ := ListTopProcesses(listLimit, uint32(os.Getpid()), anomalyLimitMB)
		stats.TopProcesses = procs
	}
	stats.Elevated = IsElevated()
	stats.TimestampUnix = time.Now().Unix()
	return stats, nil
}
