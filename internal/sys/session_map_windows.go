//go:build windows

package sys

import "unsafe"

func readSessionMapNT() map[uint32]uint32 {
	const sysProcInfoClass = 5
	buf := make([]byte, 512*1024)
	var retLen uint32
	for attempt := 0; attempt < 3; attempt++ {
		st, _, _ := procNtQuerySystemInformation.Call(
			sysProcInfoClass,
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(len(buf)),
			uintptr(unsafe.Pointer(&retLen)),
		)
		if uint32(st) == 0 {
			break
		}
		if retLen > uint32(len(buf)) && retLen <= 32*1024*1024 {
			buf = make([]byte, retLen+65536)
			continue
		}
		return nil
	}
	out := make(map[uint32]uint32, 256)
	off := 0
	for off+104 <= len(buf) {
		nextOff := *(*uint32)(unsafe.Pointer(&buf[off]))
		pid := *(*uintptr)(unsafe.Pointer(&buf[off+80]))
		sid := *(*uint32)(unsafe.Pointer(&buf[off+100]))
		out[uint32(pid)] = sid
		if nextOff == 0 {
			break
		}
		off += int(nextOff)
	}
	return out
}

func getSessionID(pid uint32, ntSessions map[uint32]uint32) uint32 {
	if pid <= 4 {
		return 0
	}
	if sid, ok := ntSessions[pid]; ok {
		return sid
	}
	var sid uint32
	r, _, _ := procProcessIdToSessionId.Call(uintptr(pid), uintptr(unsafe.Pointer(&sid)))
	if r == 0 {
		return 0
	}
	return sid
}
