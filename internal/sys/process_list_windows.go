//go:build windows

package sys

import (
	"sort"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

// ListTopProcesses retorna processos ordenados por RAM fisica (Working Set); limit <= 0 retorna todos os processos.
func ListTopProcesses(limit int, selfPID uint32, anomalyLimitMB int) ([]domain.ProcessInfo, error) {
	snaps, deskOK, err := CollectProcessSnapshots()
	if err != nil {
		return nil, err
	}
	byPID := make(map[uint32]domain.ProcSnapshot, len(snaps))
	for _, s := range snaps {
		byPID[s.PID] = s
	}
	decisions := domain.ClassifyProcesses(snaps, selfPID, deskOK, anomalyLimitMB)
	list := make([]domain.ProcessInfo, 0, len(snaps))
	for _, s := range snaps {
		if s.PID <= 4 || (s.WorkingSetBytes == 0 && s.PrivateBytes == 0) {
			continue
		}
		dec := decisions[s.PID]
		list = append(list, domain.ProcessInfo{
			PID:             s.PID,
			Name:            s.Name,
			PrivateBytes:    s.PrivateBytes,
			WorkingSetBytes: s.WorkingSetBytes,
			Protected:       dec.Action == domain.ProcKeepSystem,
			OpenWindow:      dec.Action != domain.ProcKill && (s.HasOpenWindow || dec.Reason == "janela aberta ou árvore ativa" || dec.Action == domain.ProcKeepAndTrim),
			Killable:        dec.Action == domain.ProcKill,
			Anomaly:         domain.IsAnomalyProcess(s.WorkingSetBytes, anomalyLimitMB) || domain.IsAnomalyProcess(s.PrivateBytes, anomalyLimitMB),
			Description:     domain.DescribeProcess(s, byPID, dec),
			NetProtected:    dec.Reason == "conexão de rede ativa preservada",
		})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].WorkingSetBytes != list[j].WorkingSetBytes {
			return list[i].WorkingSetBytes > list[j].WorkingSetBytes
		}
		return list[i].PrivateBytes > list[j].PrivateBytes
	})
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}
