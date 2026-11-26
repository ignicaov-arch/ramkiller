package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode"
)

const maxConfigBytes = 65536

// Config armazena todas as preferencias persistidas do RamKiller.
type Config struct {
	AlwaysOnTop               bool        `json:"alwaysOnTop"`
	LoadOnStartup             bool        `json:"loadOnStartup"`
	StartMinimized            bool        `json:"startMinimized"`
	ConfirmClean              bool        `json:"confirmClean"`
	SkipUAC                   bool        `json:"skipUac"`
	CleanMask                 CleanRegion `json:"cleanMask"`
	AutoCleanThresholdEnabled bool        `json:"autoCleanThresholdEnabled"`
	AutoCleanThresholdPct     int         `json:"autoCleanThresholdPct"`
	AutoCleanIntervalEnabled  bool        `json:"autoCleanIntervalEnabled"`
	AutoCleanIntervalMin      int         `json:"autoCleanIntervalMin"`
	AllowStandbyInAuto        bool        `json:"allowStandbyInAuto"`
	HotkeyEnabled             bool        `json:"hotkeyEnabled"`
	HotkeyModifiers           uint32      `json:"hotkeyModifiers"`
	HotkeyKey                 uint32      `json:"hotkeyKey"`
	TrayTransparent           bool        `json:"trayTransparent"`
	TrayShowBorder            bool        `json:"trayShowBorder"`
	TrayRoundCorners          bool        `json:"trayRoundCorners"`
	TrayChangeBg              bool        `json:"trayChangeBg"`
	TrayAntialias             bool        `json:"trayAntialias"`
	TrayFontName              string      `json:"trayFontName"`
	TrayFontSize              int         `json:"trayFontSize"`
	ColorText                 uint32      `json:"colorText"`
	ColorBg                   uint32      `json:"colorBg"`
	ColorWarning              uint32      `json:"colorWarning"`
	ColorDanger               uint32      `json:"colorDanger"`
	WarningLevelPct           int         `json:"warningLevelPct"`
	DangerLevelPct            int         `json:"dangerLevelPct"`
	TraySingleClick           TrayAction  `json:"traySingleClick"`
	TrayMiddleClick           TrayAction  `json:"trayMiddleClick"`
	ShowBalloonResult         bool        `json:"showBalloonResult"`
	NotificationSound         bool        `json:"notificationSound"`
	LogResults                bool        `json:"logResults"`
	AnomalyLimitMB            int         `json:"anomalyLimitMb"`
}

// DefaultConfig retorna a configuracao enxuta e segura de fabrica.
func DefaultConfig() Config {
	return Config{
		CleanMask:             MaskDefault,
		AutoCleanThresholdPct: 80,
		AutoCleanIntervalMin:  15,
		HotkeyModifiers:       0x0002,
		HotkeyKey:             0x70,
		TrayRoundCorners:      true,
		TrayChangeBg:          true,
		TrayAntialias:         true,
		TrayFontName:          "Segoe UI",
		TrayFontSize:          10,
		ColorText:             0xFFFFFF,
		ColorBg:               0x10B981,
		ColorWarning:          0xF59E0B,
		ColorDanger:           0xEF4444,
		WarningLevelPct:       60,
		DangerLevelPct:        90,
		TraySingleClick:       ActionToggleWindow,
		TrayMiddleClick:       ActionCleanMemory,
		ShowBalloonResult:     true,
		LogResults:            true,
		AnomalyLimitMB:        2560,
	}
}

func clampInt(val, minVal, maxVal int) int {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

func sanitizeFontName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || len(trimmed) > 48 {
		return "Segoe UI"
	}
	for _, r := range trimmed {
		if unicode.IsControl(r) {
			return "Segoe UI"
		}
	}
	return trimmed
}

// Sanitize normaliza invariantes numericos, mascaras, cores e acoes de bandeja.
func (c *Config) Sanitize() {
	c.CleanMask &= MaskAll
	if c.CleanMask == 0 {
		c.CleanMask = MaskDefault
	}
	c.AutoCleanThresholdPct = clampInt(c.AutoCleanThresholdPct, 10, 99)
	c.AutoCleanIntervalMin = clampInt(c.AutoCleanIntervalMin, 1, 1440)
	c.WarningLevelPct = clampInt(c.WarningLevelPct, 10, 94)
	minDanger := c.WarningLevelPct + 5
	if minDanger > 99 {
		minDanger = 99
	}
	c.DangerLevelPct = clampInt(c.DangerLevelPct, minDanger, 99)
	c.TrayFontSize = clampInt(c.TrayFontSize, 7, 16)
	c.TrayFontName = sanitizeFontName(c.TrayFontName)
	c.ColorText &= 0x00FFFFFF
	c.ColorBg &= 0x00FFFFFF
	c.ColorWarning &= 0x00FFFFFF
	c.ColorDanger &= 0x00FFFFFF
	if c.TraySingleClick < ActionToggleWindow || c.TraySingleClick > ActionOpenTaskMgr {
		c.TraySingleClick = ActionToggleWindow
	}
	if c.TrayMiddleClick < ActionToggleWindow || c.TrayMiddleClick > ActionOpenTaskMgr {
		c.TrayMiddleClick = ActionCleanMemory
	}
	c.AnomalyLimitMB = clampInt(c.AnomalyLimitMB, 512, 32768)
	if c.HotkeyKey == 0 {
		c.HotkeyModifiers = 0x0002
		c.HotkeyKey = 0x70
	}
}

// ParseConfigJSON decodifica configuracao JSON estrita rejeitando campos desconhecidos.
func ParseConfigJSON(data []byte) (Config, error) {
	if len(data) == 0 {
		return DefaultConfig(), errors.New("configuracao vazia")
	}
	if len(data) > maxConfigBytes {
		return DefaultConfig(), errors.New("arquivo de configuracao excede 64 KB")
	}
	cfg := DefaultConfig()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return DefaultConfig(), err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return DefaultConfig(), errors.New("conteudo extra inesperado apos JSON")
	}
	cfg.Sanitize()
	return cfg, nil
}
