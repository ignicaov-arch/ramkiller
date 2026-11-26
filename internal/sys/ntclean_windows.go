//go:build windows

package sys

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const (
	sysMemoryListInfoClass      = 80
	sysFileCacheInfoExClass     = 81
	sysCombinePhysMemInfoClass  = 130
	sysRegistryReconcileClass   = 155

	memCmdEmptyWorkingSets           uint32 = 2
	memCmdFlushModifiedList          uint32 = 3
	memCmdPurgeStandbyList           uint32 = 4
	memCmdPurgeLowPriorityStandby    uint32 = 5

	driveFixed          = 3
	genericWrite        = 0x40000000
	fileShareReadWrite  = 0x00000001 | 0x00000002
	openExisting        = 3
)

type memoryCombineInfoEx struct {
	Handle        uintptr
	PagesCombined uintptr
	Flags         uint32
	_             uint32
}

func ntMemoryListCommand(cmd uint32) uint32 {
	st, _, _ := procNtSetSystemInformation.Call(
		sysMemoryListInfoClass,
		uintptr(unsafe.Pointer(&cmd)),
		unsafe.Sizeof(cmd),
	)
	return uint32(st)
}

func flushSystemFileCache() uint32 {
	maxVal := ^uintptr(0)
	r, _, _ := procSetSystemFileCacheSize.Call(maxVal, maxVal, 0)
	if r != 0 {
		return 0
	}
	var sfci systemFileCacheInfo
	sfci.MinimumWorkingSet = maxVal
	sfci.MaximumWorkingSet = maxVal
	st, _, _ := procNtSetSystemInformation.Call(
		sysFileCacheInfoExClass,
		uintptr(unsafe.Pointer(&sfci)),
		unsafe.Sizeof(sfci),
	)
	return uint32(st)
}

func flushFixedVolumes() int {
	drivesMask, _, _ := procGetLogicalDrives.Call()
	flushed := 0
	for i := 0; i < 26; i++ {
		if drivesMask&(1<<uint(i)) == 0 {
			continue
		}
		letter := string(rune('A' + i))
		rootPtr, err := syscall.UTF16PtrFromString(letter + `:\`)
		if err != nil {
			continue
		}
		dtype, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(rootPtr)))
		if dtype != driveFixed {
			continue
		}
		volPtr, err := syscall.UTF16PtrFromString(`\\.\` + letter + `:`)
		if err != nil {
			continue
		}
		h, _, _ := procCreateFileW.Call(
			uintptr(unsafe.Pointer(volPtr)),
			genericWrite,
			fileShareReadWrite,
			0,
			openExisting,
			0,
			0,
		)
		if h != 0 && h != invalidHandleValue {
			ok, _, _ := procFlushFileBuffers.Call(h)
			if ok != 0 {
				flushed++
			}
			procCloseHandle.Call(h)
		}
	}
	return flushed
}

// RunKernelClean executa limpeza profunda NT Kernel nas regioes especificadas pela mascara.
func RunKernelClean(mask domain.CleanRegion) (executed bool, warnings []string) {
	if !IsElevated() {
		return false, []string{"Modo Kernel requer elevacao de Administrador (limpeza em modo usuario executada)."}
	}
	var successes int

	runCmd := func(label string, fn func() uint32) {
		if st := fn(); st == 0 {
			successes++
		} else {
			warnings = append(warnings, fmt.Sprintf("%s retornou NTSTATUS 0x%08X", label, st))
		}
	}

	if mask&domain.RegionWorkingSet != 0 {
		runCmd("WorkingSet(Kernel)", func() uint32 { return ntMemoryListCommand(memCmdEmptyWorkingSets) })
	}
	if mask&domain.RegionSystemFileCache != 0 {
		runCmd("SystemFileCache", flushSystemFileCache)
	}
	if mask&domain.RegionModifiedFileCache != 0 {
		if flushFixedVolumes() > 0 {
			successes++
		}
	}
	if mask&domain.RegionModifiedPageList != 0 {
		runCmd("ModifiedPageList", func() uint32 { return ntMemoryListCommand(memCmdFlushModifiedList) })
	}
	if mask&domain.RegionStandbyList != 0 {
		runCmd("StandbyList", func() uint32 { return ntMemoryListCommand(memCmdPurgeStandbyList) })
	}
	if mask&domain.RegionStandbyPriority0 != 0 {
		runCmd("StandbyPriority0", func() uint32 { return ntMemoryListCommand(memCmdPurgeLowPriorityStandby) })
	}
	if mask&domain.RegionRegistryCache != 0 {
		runCmd("RegistryCache", func() uint32 {
			st, _, _ := procNtSetSystemInformation.Call(sysRegistryReconcileClass, 0, 0)
			return uint32(st)
		})
	}
	if mask&domain.RegionCombineMemoryLists != 0 {
		runCmd("CombineMemoryLists", func() uint32 {
			var info memoryCombineInfoEx
			st, _, _ := procNtSetSystemInformation.Call(
				sysCombinePhysMemInfoClass,
				uintptr(unsafe.Pointer(&info)),
				unsafe.Sizeof(info),
			)
			return uint32(st)
		})
	}
	return successes > 0, warnings
}
