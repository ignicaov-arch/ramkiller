//go:build windows

package sys

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
	"unsafe"
)

const sysTimeOfDayInfoClass = 3

// SessionState registra a pausa voluntaria ate o proximo reinicio do Windows.
type SessionState struct {
	PausedBootID int64 `json:"pausedBootId"`
}

// GetBootSessionID retorna o timestamp de boot do kernel Windows (identificador unico por reinicio).
func GetBootSessionID() int64 {
	var buf [48]byte
	var retLen uint32
	st, _, _ := procNtQuerySystemInformation.Call(
		sysTimeOfDayInfoClass,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&retLen)),
	)
	if uint32(st) == 0 {
		boot := *(*int64)(unsafe.Pointer(&buf[0]))
		if boot > 0 {
			return boot
		}
	}
	r0, _, _ := procGetTickCount64.Call()
	uptimeSec := int64(uint64(r0) / 1000)
	return (time.Now().Unix() - uptimeSec) / 60
}

func sessionFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.json"), nil
}

func readSessionState(path string) (SessionState, error) {
	f, err := os.Open(path)
	if err != nil {
		return SessionState{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 4096))
	if err != nil {
		return SessionState{}, err
	}
	var st SessionState
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&st); err != nil {
		return SessionState{}, err
	}
	return st, nil
}

func writeSessionState(path string, st SessionState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func isPausedForBoot(path string, bootID int64) bool {
	if bootID == 0 {
		return false
	}
	st, err := readSessionState(path)
	if err != nil {
		return false
	}
	return st.PausedBootID == bootID
}

func pauseBootAt(path string, bootID int64) error {
	return writeSessionState(path, SessionState{PausedBootID: bootID})
}

// IsPausedForCurrentBoot informa se o usuario encerrou o RamKiller ate reiniciar o PC.
func IsPausedForCurrentBoot() bool {
	p, err := sessionFilePath()
	if err != nil {
		return false
	}
	return isPausedForBoot(p, GetBootSessionID())
}

// PauseUntilReboot grava pausa vinculada ao boot atual e sinaliza parada ao guardiao.
func PauseUntilReboot() error {
	p, err := sessionFilePath()
	if err != nil {
		return err
	}
	if err := pauseBootAt(p, GetBootSessionID()); err != nil {
		return err
	}
	SignalGuardianStop()
	return nil
}

// ClearPauseState remove a trava de pausa quando o usuario reabre manualmente o app.
func ClearPauseState() error {
	p, err := sessionFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
