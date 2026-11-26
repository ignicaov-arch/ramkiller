//go:build windows

package sys

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ignicaov-arch/ramkiller/internal/domain"
)

const (
	maxConfigFileBytes = 65536
	maxLogFileBytes    = 2 * 1024 * 1024
)

// ConfigDir retorna %APPDATA%\RamKiller criando a pasta com permissao restrita.
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		home, hErr := os.UserHomeDir()
		if hErr != nil {
			return "", fmt.Errorf("config dir indisponivel: %w", hErr)
		}
		base = filepath.Join(home, "AppData", "Roaming")
	}
	dir := filepath.Join(base, "RamKiller")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func configFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig carrega a configuracao validada ou devolve DefaultConfig se ausente.
func LoadConfig() (domain.Config, error) {
	p, err := configFilePath()
	if err != nil {
		return domain.DefaultConfig(), err
	}
	return LoadConfigFromPath(p)
}

// LoadConfigFromPath carrega configuracao estrita de um caminho especifico.
func LoadConfigFromPath(path string) (domain.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.DefaultConfig(), nil
		}
		return domain.DefaultConfig(), err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxConfigFileBytes+1))
	if err != nil {
		return domain.DefaultConfig(), err
	}
	return domain.ParseConfigJSON(data)
}

// SaveConfig persiste a configuracao de forma atomica em %APPDATA%\RamKiller\config.json.
func SaveConfig(cfg domain.Config) error {
	p, err := configFilePath()
	if err != nil {
		return err
	}
	return SaveConfigToPath(p, cfg)
}

// SaveConfigToPath grava configuracao sanitizada usando arquivo temporario e rename atomico.
func SaveConfigToPath(path string, cfg domain.Config) error {
	cfg.Sanitize()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// AppendCleanLog registra o resultado da limpeza em log rotacionado (<= 2 MB).
func AppendCleanLog(res domain.CleanResult) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	logPath := filepath.Join(dir, "ramkiller.log")
	if info, err := os.Stat(logPath); err == nil && info.Size() >= maxLogFileBytes {
		_ = os.Remove(logPath + ".old")
		_ = os.Rename(logPath, logPath+".old")
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	line := fmt.Sprintf(
		"[%s] src=%s elevated=%t kernel=%t killed=%d procs=%d freed=%s dur=%dms\n",
		time.Now().Format("2006-01-02 15:04:05"),
		res.Source,
		res.Elevated,
		res.KernelExecuted,
		res.KilledProcs,
		res.TrimmedProcs,
		domain.FormatBytes(res.FreedBytes),
		res.DurationMs,
	)
	_, err = f.WriteString(line)
	return err
}
