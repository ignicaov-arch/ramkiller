//go:build windows

package sys

import "unsafe"

const (
	afInet                    = 2
	afInet6                   = 23
	tcpTableOwnerPIDConn      = 4
	tcpStateSynSent           = 3
	tcpStateEstablished       = 5
	errInsufficientBufferWin  = 122
)

type mibTcpRowOwnerPID struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPid  uint32
}

type mibTcp6RowOwnerPID struct {
	LocalAddr     [16]byte
	LocalScopeID  uint32
	LocalPort     uint32
	RemoteAddr    [16]byte
	RemoteScopeID uint32
	RemotePort    uint32
	State         uint32
	OwningPid     uint32
}

func isActiveTCPState(st uint32) bool {
	return st == tcpStateEstablished || st == tcpStateSynSent
}

func isLoopbackV4(addr uint32) bool {
	return addr == 0 || (addr&0xFF) == 127
}

func isLoopbackV6(addr [16]byte) bool {
	var allZero = true
	for i := 0; i < 16; i++ {
		if addr[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return true
	}
	isV6Loop := true
	for i := 0; i < 15; i++ {
		if addr[i] != 0 {
			isV6Loop = false
			break
		}
	}
	if isV6Loop && addr[15] == 1 {
		return true
	}
	// IPv4-mapped IPv6 (::ffff:127.x.x.x or ::ffff:0.0.0.0)
	for i := 0; i < 10; i++ {
		if addr[i] != 0 {
			return false
		}
	}
	if addr[10] == 0xff && addr[11] == 0xff {
		if addr[12] == 127 || (addr[12] == 0 && addr[13] == 0 && addr[14] == 0 && addr[15] == 0) {
			return true
		}
	}
	return false
}

func queryTCPTablePIDs(family uint32, out map[uint32]bool) {
	size := uint32(16384)
	for attempt := 0; attempt < 3; attempt++ {
		buf := make([]byte, size)
		r, _, _ := procGetExtendedTcpTable.Call(
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&size)),
			0,
			uintptr(family),
			tcpTableOwnerPIDConn,
			0,
		)
		if r == errInsufficientBufferWin && size > 4 && size <= 16*1024*1024 {
			continue
		}
		if r != 0 || len(buf) < 4 {
			return
		}
		count := int(*(*uint32)(unsafe.Pointer(&buf[0])))
		if family == afInet {
			rowSize := int(unsafe.Sizeof(mibTcpRowOwnerPID{}))
			for i := 0; i < count && 4+(i+1)*rowSize <= len(buf); i++ {
				row := (*mibTcpRowOwnerPID)(unsafe.Pointer(&buf[4+i*rowSize]))
				if row.OwningPid > 4 && isActiveTCPState(row.State) && !isLoopbackV4(row.RemoteAddr) {
					out[row.OwningPid] = true
				}
			}
		} else {
			rowSize := int(unsafe.Sizeof(mibTcp6RowOwnerPID{}))
			for i := 0; i < count && 4+(i+1)*rowSize <= len(buf); i++ {
				row := (*mibTcp6RowOwnerPID)(unsafe.Pointer(&buf[4+i*rowSize]))
				if row.OwningPid > 4 && isActiveTCPState(row.State) && !isLoopbackV6(row.RemoteAddr) {
					out[row.OwningPid] = true
				}
			}
		}
		return
	}
}

// ActiveNetworkPIDs retorna PIDs com conexoes TCP externas ativas (ESTABLISHED ou SYN_SENT).
func ActiveNetworkPIDs() map[uint32]bool {
	out := make(map[uint32]bool, 32)
	queryTCPTablePIDs(afInet, out)
	queryTCPTablePIDs(afInet6, out)
	return out
}
