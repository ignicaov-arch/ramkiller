//go:build windows

package sys

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

func TestSystem32PathAntiHijacking(t *testing.T) {
	p := System32Path(`../../evil/psapi.dll`)
	if !strings.HasSuffix(strings.ToLower(p), `\system32\psapi.dll`) {
		t.Fatalf("caminho System32 inesperado (risco CWE-427): %s", p)
	}
	if !filepath.IsAbs(p) {
		t.Fatalf("esperava caminho absoluto, obteve: %s", p)
	}
}

func TestReadMemoryStatsReal(t *testing.T) {
	stats, err := ReadMemoryStats(5, 2560)
	if err != nil {
		t.Fatalf("ReadMemoryStats falhou: %v", err)
	}
	if stats.Physical.TotalBytes == 0 || stats.Physical.Percent < 1 || stats.Physical.Percent > 100 {
		t.Fatalf("metricas fisicas invalidas: %+v", stats.Physical)
	}
	if stats.Pagefile.TotalBytes == 0 {
		t.Errorf("pagefile total nao deveria ser zero")
	}
	if len(stats.TopProcesses) == 0 || len(stats.TopProcesses) > 5 {
		t.Errorf("TopProcesses inesperado: len=%d", len(stats.TopProcesses))
	}
	for i := 1; i < len(stats.TopProcesses); i++ {
		if stats.TopProcesses[i-1].PrivateBytes < stats.TopProcesses[i].PrivateBytes {
			t.Errorf("TopProcesses nao ordenado por PrivateBytes")
		}
	}
}

func TestTrimSelfAndEngineConcurrency(t *testing.T) {
	if err := TrimProcessWorkingSet(uint32(os.Getpid())); err != nil {
		t.Fatalf("TrimProcessWorkingSet no proprio processo falhou: %v", err)
	}

	eng := NewEngine()
	cfg := domain.DefaultConfig()
	cfg.LogResults = false

	var wg sync.WaitGroup
	var okCount, busyCount int
	var mu sync.Mutex
	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := eng.Clean(context.Background(), domain.SourceManual, cfg, domain.RegionWorkingSet)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				okCount++
				if res.TrimmedProcs < 1 {
					t.Errorf("esperava ao menos 1 processo reciclado, obteve %d", res.TrimmedProcs)
				}
			} else if errors.Is(err, ErrCleanInProgress) {
				busyCount++
			} else {
				t.Errorf("erro inesperado na limpeza concorrente: %v", err)
			}
		}()
	}
	wg.Wait()
	if okCount == 0 {
		t.Fatalf("ao menos uma execucao deveria completar, ok=%d busy=%d", okCount, busyCount)
	}
}

func TestConfigStoreRoundtripAndHostile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "config.json")
	cfg := domain.DefaultConfig()
	cfg.AutoCleanThresholdPct = 77
	cfg.TrayShowBorder = true

	if err := SaveConfigToPath(tmp, cfg); err != nil {
		t.Fatalf("SaveConfigToPath falhou: %v", err)
	}
	loaded, err := LoadConfigFromPath(tmp)
	if err != nil {
		t.Fatalf("LoadConfigFromPath falhou: %v", err)
	}
	if loaded.AutoCleanThresholdPct != 77 || !loaded.TrayShowBorder {
		t.Fatalf("config carregada diverge: %+v", loaded)
	}

	if err := os.WriteFile(tmp, []byte(`{"rogue":123}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfigFromPath(tmp); err == nil {
		t.Error("esperava rejeicao para campo desconhecido no config")
	}
}
