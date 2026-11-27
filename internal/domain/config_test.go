package domain

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDefaultConfigAndSanitizeBounds(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ConfigVersion != CurrentConfigVersion || cfg.CleanMask != MaskAll {
		t.Fatalf("defaults v2 inesperados: ver=%d mask=%v", cfg.ConfigVersion, cfg.CleanMask)
	}
	if !cfg.LoadOnStartup || !cfg.StartMenuShortcut || !cfg.SelfHeal || !cfg.KillBackgroundApps || !cfg.AllowStandbyInAuto || !cfg.SkipUAC {
		t.Fatalf("flags principais deveriam iniciar ativas: %+v", cfg)
	}
	if cfg.AutoCleanThresholdPct != 80 || cfg.WarningLevelPct != 60 || cfg.DangerLevelPct != 90 {
		t.Fatalf("limiares padrao inesperados: %+v", cfg)
	}

	dirty := Config{
		CleanMask:             CleanRegion(0xFFFF0000),
		AutoCleanThresholdPct: 3,
		AutoCleanIntervalMin:  9999,
		WarningLevelPct:       95,
		DangerLevelPct:        20,
		TrayFontName:          "Segoe\x00UI",
		TrayFontSize:          42,
		ColorText:             0xFF112233,
		TraySingleClick:       TrayAction(99),
		TrayMiddleClick:       TrayAction(-1),
		AnomalyLimitMB:        100,
	}
	dirty.Sanitize()

	if dirty.CleanMask != RegionWorkingSet {
		t.Errorf("esperava fallback RegionWorkingSet, obteve %v", dirty.CleanMask)
	}
	if dirty.AutoCleanThresholdPct != 10 || dirty.AutoCleanIntervalMin != 1440 {
		t.Errorf("limites nao aplicados: thr=%d int=%d", dirty.AutoCleanThresholdPct, dirty.AutoCleanIntervalMin)
	}
	if dirty.WarningLevelPct != 94 || dirty.DangerLevelPct != 99 {
		t.Errorf("warning/danger inconsistentes: %d/%d", dirty.WarningLevelPct, dirty.DangerLevelPct)
	}
	if dirty.TrayFontName != "Segoe UI" || dirty.TrayFontSize != 16 {
		t.Errorf("fonte nao sanitizada: %q %d", dirty.TrayFontName, dirty.TrayFontSize)
	}
	if dirty.ColorText != 0x00112233 {
		t.Errorf("mascara RGB falhou: %X", dirty.ColorText)
	}
	if dirty.TraySingleClick != ActionToggleWindow || dirty.TrayMiddleClick != ActionCleanMemory {
		t.Errorf("acoes invalidas nao corrigidas: %d/%d", dirty.TraySingleClick, dirty.TrayMiddleClick)
	}
	if dirty.AnomalyLimitMB != 512 {
		t.Errorf("AnomalyLimitMB nao corrigido: %d", dirty.AnomalyLimitMB)
	}
}

func TestParseConfigJSONHostileInputsAndV1Upgrade(t *testing.T) {
	valid, err := json.Marshal(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseConfigJSON(valid)
	if err != nil || parsed.AutoCleanThresholdPct != 80 || parsed.CleanMask != MaskAll {
		t.Fatalf("falhou ao ler JSON valido: %v", err)
	}

	v1Legacy := []byte(`{"cleanMask":231,"loadOnStartup":false,"allowStandbyInAuto":false}`)
	upgraded, err := ParseConfigJSON(v1Legacy)
	if err != nil {
		t.Fatalf("falhou ao migrar v1: %v", err)
	}
	if upgraded.ConfigVersion != CurrentConfigVersion || upgraded.CleanMask != MaskAll || !upgraded.LoadOnStartup || !upgraded.AllowStandbyInAuto {
		t.Fatalf("upgrade v1->v3 incompleto: %+v", upgraded)
	}

	v2Legacy := []byte(`{"configVersion":2,"cleanMask":231,"allowStandbyInAuto":false}`)
	upgradedV2, err := ParseConfigJSON(v2Legacy)
	if err != nil || upgradedV2.ConfigVersion != CurrentConfigVersion || upgradedV2.CleanMask != MaskAll || !upgradedV2.AllowStandbyInAuto || !upgradedV2.StartMenuShortcut {
		t.Fatalf("upgrade v2->v4 incompleto: err=%v cfg=%+v", err, upgradedV2)
	}
	v3Legacy := []byte(`{"configVersion":3,"loadOnStartup":true,"cleanMask":255}`)
	upgradedV3, err := ParseConfigJSON(v3Legacy)
	if err != nil || upgradedV3.ConfigVersion != CurrentConfigVersion || !upgradedV3.StartMenuShortcut {
		t.Fatalf("upgrade v3->v4 incompleto: err=%v cfg=%+v", err, upgradedV3)
	}

	if _, err := ParseConfigJSON(nil); err == nil {
		t.Error("esperava erro para JSON vazio")
	}
	if _, err := ParseConfigJSON([]byte(`{"unknownField":true}`)); err == nil {
		t.Error("esperava erro para campo desconhecido")
	}
	if _, err := ParseConfigJSON([]byte(`{"alwaysOnTop":true}{"extra":1}`)); err == nil {
		t.Error("esperava erro para trailing JSON")
	}
	oversized := bytes.Repeat([]byte(" "), maxConfigBytes+10)
	if _, err := ParseConfigJSON(oversized); err == nil {
		t.Error("esperava erro para payload acima de 64 KB")
	}
	badFont := []byte(`{"trayFontName":"   "}`)
	c2, err := ParseConfigJSON(badFont)
	if err != nil || !strings.EqualFold(c2.TrayFontName, "Segoe UI") {
		t.Errorf("esperava fallback de fonte: %+v err=%v", c2, err)
	}
}
