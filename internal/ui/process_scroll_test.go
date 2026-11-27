//go:build windows

package ui

import (
	"fmt"
	"testing"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

func TestFullProcessListVirtualizationAndScroll(t *testing.T) {
	procs := make([]domain.ProcessInfo, 30)
	for i := range procs {
		procs[i] = domain.ProcessInfo{
			PID:             uint32(1000 + i),
			Name:            fmt.Sprintf("app%02d.exe", i),
			Description:     "WhatsApp · Interface WebView2",
			WorkingSetBytes: uint64(300-i*5) * 1024 * 1024,
			Killable:        i%2 == 1,
			OpenWindow:      i%2 == 0,
		}
	}

	ctrl := &AppController{
		cfg:   domain.DefaultConfig(),
		stats: domain.MemoryStats{Elevated: true, TopProcesses: procs},
	}

	c0, hits0 := RenderMainCanvas(ctrl.stats, ctrl.cfg, "", "", false, 0)
	c0.Close()
	if len(hits0) == 0 {
		t.Fatal("esperava hit regions no painel principal")
	}

	layout := computeProcListLayout(len(procs), 0, 16, mainWinWidth-32)
	if layout.MaxScroll <= 0 || layout.ThumbRect.Bottom <= layout.ThumbRect.Top {
		t.Fatalf("esperava barra de rolagem ativa para 30 processos: %+v", layout)
	}
	ctrl.procMaxScroll = layout.MaxScroll
	ctrl.scrollTrack = layout.TrackRect
	ctrl.scrollThumb = layout.ThumbRect

	// Roda do mouse desce a lista
	wheel := int16(-240)
	negDelta := uintptr(uint32(uint16(wheel)) << 16)
	ctrl.handleMainWheel(negDelta)
	if ctrl.procScrollY <= 0 {
		t.Errorf("esperava procScrollY > 0 apos WM_MOUSEWHEEL para baixo, obteve %d", ctrl.procScrollY)
	}

	// Home volta ao topo e End vai ao fim
	ctrl.handleMainKeyDown(VK_END)
	if ctrl.procScrollY != ctrl.procMaxScroll {
		t.Errorf("VK_END deveria ir para %d, obteve %d", ctrl.procMaxScroll, ctrl.procScrollY)
	}
	ctrl.handleMainKeyDown(VK_HOME)
	if ctrl.procScrollY != 0 {
		t.Errorf("VK_HOME deveria zerar scroll, obteve %d", ctrl.procScrollY)
	}

	// Renderiza perto do final e confirma ultimo processo visivel nos hits
	cEnd, hitsEnd := RenderMainCanvas(ctrl.stats, ctrl.cfg, "", "", false, ctrl.procMaxScroll)
	cEnd.Close()
	foundLast := false
	for _, h := range hitsEnd {
		if h.ID == "trim_pid_1029" {
			foundLast = true
		}
	}
	if !foundLast {
		t.Error("ultimo processo da lista (PID 1029) deveria estar clicavel ao rolar ate o final")
	}
}
