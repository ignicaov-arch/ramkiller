//go:build windows

package sys

import (
	"os"
	"testing"
	"time"
)

func TestActiveNetworkPIDsFastBatchTrimAndZeroTopTelemetry(t *testing.T) {
	netPIDs := ActiveNetworkPIDs()
	if netPIDs == nil {
		t.Fatal("ActiveNetworkPIDs retornou mapa nulo")
	}
	for pid := range netPIDs {
		if pid <= 4 {
			t.Fatalf("PID de nucleo (%d) nao deve constar como socket de usuario", pid)
		}
	}

	start := time.Now()
	if err := TrimProcessWorkingSet(uint32(os.Getpid())); err != nil {
		t.Fatalf("TrimProcessWorkingSet falhou: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("TrimProcessWorkingSet em lote demorou %v (esperado < 100ms sem sleep)", elapsed)
	}

	fastStats, err := ReadMemoryStats(0, 2560)
	if err != nil {
		t.Fatalf("ReadMemoryStats(0) falhou: %v", err)
	}
	if len(fastStats.TopProcesses) != 0 || fastStats.Physical.TotalBytes == 0 {
		t.Errorf("ReadMemoryStats(0) inesperado: top=%d total=%d", len(fastStats.TopProcesses), fastStats.Physical.TotalBytes)
	}
}
