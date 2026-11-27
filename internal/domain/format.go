package domain

import (
	"fmt"
	"strings"
	"unicode/utf16"
)

const (
	kb = 1024
	mb = 1024 * kb
	gb = 1024 * mb
)

// FormatBytes formata bytes de forma compacta para paineis e tooltips.
func FormatBytes(b uint64) string {
	switch {
	case b >= gb:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.0f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.0f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// SeverityForPercent classifica a porcentagem de uso em Normal, Alerta ou Perigo.
func SeverityForPercent(pct int, cfg Config) Severity {
	if pct >= cfg.DangerLevelPct {
		return SeverityDanger
	}
	if pct >= cfg.WarningLevelPct {
		return SeverityWarning
	}
	return SeverityNormal
}

// TruncateUTF16 garante no maximo maxUnits unidades UTF-16 (limite szTip Win32).
func TruncateUTF16(s string, maxUnits int) string {
	if maxUnits <= 0 {
		return ""
	}
	encoded := utf16.Encode([]rune(s))
	if len(encoded) <= maxUnits {
		return s
	}
	cut := encoded[:maxUnits]
	if cut[len(cut)-1] >= 0xD800 && cut[len(cut)-1] <= 0xDBFF {
		cut = cut[:len(cut)-1]
	}
	return string(utf16.Decode(cut))
}

// FormatTooltip gera o tooltip da bandeja respeitando limite <= 120 UTF-16 code units.
func FormatTooltip(stats MemoryStats) string {
	raw := fmt.Sprintf(
		"RamKiller · Física: %d%% (%s/%s)\nPaginação: %d%% · Cache: %d%%\nLivre: %s",
		stats.Physical.Percent,
		FormatBytes(stats.Physical.UsedBytes),
		FormatBytes(stats.Physical.TotalBytes),
		stats.Pagefile.Percent,
		stats.SystemCache.Percent,
		FormatBytes(stats.Physical.FreeBytes),
	)
	return TruncateUTF16(raw, 120)
}

// FormatCleanSummary resume o resultado da limpeza unificada para UI e balao da bandeja.
func FormatCleanSummary(res CleanResult) string {
	mode := "Modo Usuário"
	if res.KernelExecuted {
		mode = "Modo Kernel"
	}
	procStats := fmt.Sprintf("%d reciclados", res.TrimmedProcs)
	if res.KilledProcs > 0 && res.TrimmedProcs > 0 {
		procStats = fmt.Sprintf("%d encerrados · %d reciclados", res.KilledProcs, res.TrimmedProcs)
	} else if res.KilledProcs > 0 {
		procStats = fmt.Sprintf("%d encerrados", res.KilledProcs)
	}
	switch {
	case res.FreedBytes > 0 && res.PagefileFreedBytes >= 1*mb:
		return fmt.Sprintf("+%s RAM · +%s Paginação (%s · %s · %d ms)",
			FormatBytes(res.FreedBytes), FormatBytes(res.PagefileFreedBytes), mode, procStats, res.DurationMs)
	case res.FreedBytes == 0 && res.PagefileFreedBytes >= 1*mb:
		return fmt.Sprintf("+%s Paginação (%s · %s · %d ms)",
			FormatBytes(res.PagefileFreedBytes), mode, procStats, res.DurationMs)
	case res.FreedBytes > 0:
		return fmt.Sprintf("+%s liberados (%s · %s · %d ms)", FormatBytes(res.FreedBytes), mode, procStats, res.DurationMs)
	default:
		return fmt.Sprintf("Limpeza concluída (%s · %s · %d ms)", mode, procStats, res.DurationMs)
	}
}

// FormatHotkeyLabel formata modificadores Win32 e codigo VK para exibicao humana.
func FormatHotkeyLabel(mods uint32, vk uint32) string {
	var parts []string
	if mods&0x0002 != 0 {
		parts = append(parts, "Ctrl")
	}
	if mods&0x0001 != 0 {
		parts = append(parts, "Alt")
	}
	if mods&0x0004 != 0 {
		parts = append(parts, "Shift")
	}
	keyName := fmt.Sprintf("0x%02X", vk)
	switch {
	case vk >= 0x70 && vk <= 0x7B:
		keyName = fmt.Sprintf("F%d", vk-0x70+1)
	case vk >= 'A' && vk <= 'Z':
		keyName = string(rune(vk))
	case vk >= '0' && vk <= '9':
		keyName = string(rune(vk))
	}
	parts = append(parts, keyName)
	return strings.Join(parts, " + ")
}
