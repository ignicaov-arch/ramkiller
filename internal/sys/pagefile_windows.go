//go:build windows

package sys

import (
	"os"
	"strings"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const minLeakyCommitBytes uint64 = 140 * 1024 * 1024

var leakyCommitHostNames = map[string]struct{}{
	"dtsapo4service.exe":          {},
	"textinputhost.exe":           {},
	"startmenuexperiencehost.exe": {},
	"shellexperiencehost.exe":     {},
	"searchapp.exe":               {},
	"phoneexperiencehost.exe":     {},
}

// RecycleLeakyPagefileHosts reinicia hosts auto-regeneraveis com vazamento alto no pagefile (>= 140 MB).
func RecycleLeakyPagefileHosts(snaps []domain.ProcSnapshot, fgPID uint32) (recycled int, names []string) {
	seen := make(map[string]struct{})
	selfPID := uint32(os.Getpid())
	for _, s := range snaps {
		if s.PID <= 4 || s.PID == selfPID || (fgPID != 0 && s.PID == fgPID) || s.HasOpenWindow {
			continue
		}
		norm := strings.ToLower(strings.TrimSpace(s.Name))
		if _, ok := leakyCommitHostNames[norm]; !ok {
			continue
		}
		if s.PrivateBytes < minLeakyCommitBytes {
			continue
		}
		if err := TerminateProcessSafe(s.PID); err == nil {
			recycled++
			if _, exists := seen[s.Name]; !exists && len(names) < 20 {
				seen[s.Name] = struct{}{}
				names = append(names, s.Name)
			}
		}
	}
	return recycled, names
}
