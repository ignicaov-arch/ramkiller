//go:build windows

package ui

import (
	"fmt"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const (
	procCardY        = 260
	procCardH        = 314
	procViewportTop  = procCardY + 36
	procViewportH    = 272
	procRowHeight    = 38
	procScrollTrackW = 8
	procMinThumbH    = 28
)

// ProcListLayout descreve metricas calculadas do viewport e barra de rolagem.
type ProcListLayout struct {
	MaxScroll int
	ScrollY   int
	TrackRect RECT
	ThumbRect RECT
}

func computeProcListLayout(totalCount, scrollY, cardX, cardW int) ProcListLayout {
	totalH := totalCount * procRowHeight
	maxScroll := 0
	if totalH > procViewportH {
		maxScroll = totalH - procViewportH
	}
	if scrollY < 0 {
		scrollY = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	layout := ProcListLayout{MaxScroll: maxScroll, ScrollY: scrollY}
	if maxScroll > 0 {
		tx := cardX + cardW - 14
		ty := procViewportTop + 4
		th := procViewportH - 8
		layout.TrackRect = RECT{int32(tx), int32(ty), int32(tx + procScrollTrackW), int32(ty + th)}
		thumbH := (th * procViewportH) / totalH
		if thumbH < procMinThumbH {
			thumbH = procMinThumbH
		}
		if thumbH > th {
			thumbH = th
		}
		usable := th - thumbH
		thumbTop := ty
		if usable > 0 {
			thumbTop += (scrollY * usable) / maxScroll
		}
		layout.ThumbRect = RECT{int32(tx), int32(thumbTop), int32(tx + procScrollTrackW), int32(thumbTop + thumbH)}
	}
	return layout
}

func blitViewport(dst *Canvas, dstX, dstY int, src *Canvas) {
	if dst == nil || src == nil || dst.HDC == 0 || src.HDC == 0 {
		return
	}
	procGdiFlush.Call()
	procBitBlt.Call(dst.HDC, uintptr(dstX), uintptr(dstY), uintptr(src.Width), uintptr(src.Height), src.HDC, 0, 0, srcCopy)
	procGdiFlush.Call()
}

func clipRectToViewport(r RECT, vpTop, vpBottom int32) (RECT, bool) {
	if r.Bottom <= vpTop || r.Top >= vpBottom {
		return RECT{}, false
	}
	if r.Top < vpTop {
		r.Top = vpTop
	}
	if r.Bottom > vpBottom {
		r.Bottom = vpBottom
	}
	if r.Bottom <= r.Top {
		return RECT{}, false
	}
	return r, true
}

func appendClippedHit(hits []HitRegion, h HitRegion, vpX, vpY int, vpBottom int32) []HitRegion {
	shifted := RECT{h.Rect.Left + int32(vpX), h.Rect.Top + int32(vpY), h.Rect.Right + int32(vpX), h.Rect.Bottom + int32(vpY)}
	clipped, ok := clipRectToViewport(shifted, int32(vpY), vpBottom)
	if !ok {
		return hits
	}
	h.Rect = clipped
	return append(hits, h)
}

func drawProcessesCardFull(c *Canvas, x, w int, procs []domain.ProcessInfo, scrollY int, hoverID string) ([]HitRegion, ProcListLayout) {
	const pad = 12
	var hits []HitRegion
	layout := computeProcListLayout(len(procs), scrollY, x, w)

	c.FillRoundRect(x, procCardY, w, procCardH, RadiusCard, ColorBgCard)
	c.StrokeRoundRect(x, procCardY, w, procCardH, RadiusCard, 1, ColorBorder)
	title := fmt.Sprintf("Processos na RAM (%d)", len(procs))
	c.DrawText(title, FontUI, 13, true, ColorTextPrimary, RECT{int32(x + pad), int32(procCardY + 8), int32(x + 250), int32(procCardY + 30)}, dtLeft|dtVCenter|dtSingleLine)
	c.DrawText("Role p/ ver todos · RAM Física", FontUI, 11, false, ColorTextMuted, RECT{int32(x + 250), int32(procCardY + 8), int32(x + w - pad), int32(procCardY + 30)}, dtRight|dtVCenter|dtSingleLine)
	c.FillRect(x+pad, procCardY+34, w-pad*2, 1, ColorBorder)

	if len(procs) == 0 {
		c.DrawText("Nenhum processo ativo listado.", FontUI, 12, false, ColorTextMuted, RECT{int32(x + pad), int32(procViewportTop + 40), int32(x + w - pad), int32(procViewportTop + 80)}, dtCenter|dtVCenter|dtSingleLine)
		return hits, layout
	}

	vpX, vpW := x+2, w-4
	vp := NewCanvas(vpW, procViewportH)
	defer vp.Close()
	vp.ClearOpaque(ColorBgCard)

	startIdx := layout.ScrollY / procRowHeight
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := (layout.ScrollY+procViewportH)/procRowHeight + 1
	if endIdx > len(procs) {
		endIdx = len(procs)
	}

	rowLeft := 10
	rowRight := vpW - 14
	if layout.MaxScroll > 0 {
		rowRight = vpW - 22
	}
	for i := startIdx; i < endIdx; i++ {
		p := procs[i]
		ry := i*procRowHeight - layout.ScrollY
		if i > 0 {
			vp.FillRect(rowLeft, ry, rowRight-rowLeft, 1, ColorBorder)
		}
		nameColor := ColorTextPrimary
		if p.Anomaly {
			nameColor = ColorAmber
		}
		label := fmt.Sprintf("%s (%d)", p.Name, p.PID)
		vp.DrawText(label, FontUI, 12, false, nameColor, RECT{int32(rowLeft), int32(ry + 3), int32(rowRight - 208), int32(ry + 21)}, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)

		desc := p.Description
		if desc == "" {
			desc = "Processo ativo"
		}
		vp.DrawText(desc, FontUI, 10, false, ColorTextMuted, RECT{int32(rowLeft), int32(ry + 20), int32(rowRight - 208), int32(ry + 35)}, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)

		memColor := ColorTextMuted
		if p.Anomaly {
			memColor = ColorRed
		}
		vp.DrawText(domain.FormatBytes(p.WorkingSetBytes), FontUI, 11, true, memColor, RECT{int32(rowRight - 204), int32(ry), int32(rowRight - 140), int32(ry + procRowHeight)}, dtRight|dtVCenter|dtSingleLine)

		trimBtn := drawButton(vp, fmt.Sprintf("trim_pid_%d", p.PID), rowRight-134, ry+7, 62, 24, "Reciclar", ColorBgElevated, ColorBorder, ColorTextPrimary, false, hoverID)
		trimBtn.Arg = p.PID
		hits = appendClippedHit(hits, trimBtn, vpX, procViewportTop, int32(procViewportTop+procViewportH))

		sx := rowRight - 66
		if p.Killable {
			killBtn := drawButton(vp, fmt.Sprintf("kill_pid_%d", p.PID), sx, ry+7, 66, 24, "Matar", ColorRedSoft, ColorRed, ColorRed, true, hoverID)
			killBtn.Arg = p.PID
			hits = appendClippedHit(hits, killBtn, vpX, procViewportTop, int32(procViewportTop+procViewportH))
		} else if p.OpenWindow {
			vp.FillRoundRect(sx, ry+7, 66, 24, 6, ColorGreenSoft)
			vp.DrawText("Aberto", FontUI, 11, false, ColorGreen, RECT{int32(sx), int32(ry + 7), int32(sx + 66), int32(ry + 31)}, dtCenter|dtVCenter|dtSingleLine)
		} else {
			vp.FillRoundRect(sx, ry+7, 66, 24, 6, ColorWhiteSoft)
			vp.DrawText("Sistema", FontUI, 11, false, ColorTextMuted, RECT{int32(sx), int32(ry + 7), int32(sx + 66), int32(ry + 31)}, dtCenter|dtVCenter|dtSingleLine)
		}
	}
	blitViewport(c, vpX, procViewportTop, vp)

	if layout.MaxScroll > 0 {
		tr := layout.TrackRect
		th := layout.ThumbRect
		c.FillRoundRect(int(tr.Left), int(tr.Top), int(tr.Right-tr.Left), int(tr.Bottom-tr.Top), 4, ColorBgBase)
		thumbColor := uint32(0xFF4B5563)
		if hoverID == "proc_scroll_thumb" {
			thumbColor = uint32(0xFF6B7280)
		}
		c.FillRoundRect(int(th.Left), int(th.Top), int(th.Right-th.Left), int(th.Bottom-th.Top), 4, thumbColor)
		hits = append(hits,
			HitRegion{ID: "proc_scroll_track", Rect: tr},
			HitRegion{ID: "proc_scroll_thumb", Rect: th},
		)
	}
	return hits, layout
}
