//go:build windows

package ui

func (a *AppController) clampProcScroll() {
	a.mu.RLock()
	n := len(a.stats.TopProcesses)
	a.mu.RUnlock()
	totalH := n * procRowHeight
	maxScroll := 0
	if totalH > procViewportH {
		maxScroll = totalH - procViewportH
	}
	a.procMaxScroll = maxScroll
	if a.procScrollY > maxScroll {
		a.procScrollY = maxScroll
	}
	if a.procScrollY < 0 {
		a.procScrollY = 0
	}
}

func (a *AppController) scrollProcessListBy(deltaPx int) {
	a.clampProcScroll()
	if a.procMaxScroll <= 0 {
		return
	}
	next := a.procScrollY + deltaPx
	if next < 0 {
		next = 0
	}
	if next > a.procMaxScroll {
		next = a.procMaxScroll
	}
	if next != a.procScrollY {
		a.procScrollY = next
		if a.mainHWND != 0 {
			procInvalidateRect.Call(a.mainHWND, 0, 0)
		}
	}
}

func (a *AppController) handleMainWheel(wParam uintptr) {
	delta := int32(int16((wParam >> 16) & 0xFFFF))
	if delta == 0 {
		return
	}
	step := -int(delta) * (procRowHeight * 2) / 120
	if step == 0 {
		if delta > 0 {
			step = -procRowHeight
		} else {
			step = procRowHeight
		}
	}
	a.scrollProcessListBy(step)
}

func (a *AppController) handleMainKeyDown(vk uintptr) {
	switch vk {
	case VK_DOWN:
		a.scrollProcessListBy(procRowHeight)
	case VK_UP:
		a.scrollProcessListBy(-procRowHeight)
	case VK_NEXT:
		a.scrollProcessListBy(procViewportH - procRowHeight)
	case VK_PRIOR:
		a.scrollProcessListBy(-(procViewportH - procRowHeight))
	case VK_HOME:
		a.scrollProcessListBy(-a.procMaxScroll - procRowHeight)
	case VK_END:
		a.scrollProcessListBy(a.procMaxScroll + procRowHeight)
	}
}

func (a *AppController) handleMainLButtonDown(x, y int32) bool {
	hit, ok := FindHit(a.mainHits, x, y)
	if !ok {
		return false
	}
	switch hit.ID {
	case "proc_scroll_thumb":
		a.scrollDragging = true
		a.scrollDragStartY = y
		a.scrollDragStartVal = a.procScrollY
		a.mainHover = "proc_scroll_thumb"
		if a.mainHWND != 0 {
			procSetCapture.Call(a.mainHWND)
			procInvalidateRect.Call(a.mainHWND, 0, 0)
		}
		return true
	case "proc_scroll_track":
		page := procViewportH - procRowHeight
		if y < a.scrollThumb.Top {
			a.scrollProcessListBy(-page)
		} else if y >= a.scrollThumb.Bottom {
			a.scrollProcessListBy(page)
		}
		return true
	}
	return false
}

func (a *AppController) handleMainScrollDrag(y int32) bool {
	if !a.scrollDragging {
		return false
	}
	a.clampProcScroll()
	trackH := int(a.scrollTrack.Bottom - a.scrollTrack.Top)
	thumbH := int(a.scrollThumb.Bottom - a.scrollThumb.Top)
	usable := trackH - thumbH
	if usable <= 0 || a.procMaxScroll <= 0 {
		return true
	}
	dy := int(y - a.scrollDragStartY)
	next := a.scrollDragStartVal + (dy*a.procMaxScroll)/usable
	if next < 0 {
		next = 0
	}
	if next > a.procMaxScroll {
		next = a.procMaxScroll
	}
	if next != a.procScrollY {
		a.procScrollY = next
		if a.mainHWND != 0 {
			procInvalidateRect.Call(a.mainHWND, 0, 0)
		}
	}
	return true
}

func (a *AppController) stopScrollDrag() bool {
	if !a.scrollDragging {
		return false
	}
	a.scrollDragging = false
	procReleaseCapture.Call()
	if a.mainHWND != 0 {
		procInvalidateRect.Call(a.mainHWND, 0, 0)
	}
	return true
}
