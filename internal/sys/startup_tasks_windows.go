//go:build windows

package sys

import (
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
)

const (
	skipUACTask     = "RamKiller_AutoStart"
	interactiveTask = "RamKiller_Interactive"
	legacyUACTask   = "RamKiller_SkipUAC"
)

func runSchtasks(args ...string) (string, error) {
	cmd := exec.Command(System32Path("schtasks.exe"), args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func encodeUTF16LE(s string) []byte {
	u16 := utf16.Encode([]rune(s))
	buf := make([]byte, 2+len(u16)*2)
	buf[0] = 0xFF
	buf[1] = 0xFE
	for i, c := range u16 {
		buf[2+i*2] = byte(c)
		buf[2+i*2+1] = byte(c >> 8)
	}
	return buf
}

func buildTaskXML(exe, args string, atLogon bool, multiPolicy string) string {
	triggerBlock := "  <Triggers />\r\n"
	if atLogon {
		triggerBlock = "  <Triggers>\r\n    <LogonTrigger>\r\n      <Enabled>true</Enabled>\r\n    </LogonTrigger>\r\n  </Triggers>\r\n"
	}
	argLine := ""
	if args != "" {
		argLine = fmt.Sprintf("\r\n      <Arguments>%s</Arguments>", html.EscapeString(args))
	}
	return "<?xml version=\"1.0\" encoding=\"UTF-16\"?>\r\n" +
		"<Task version=\"1.2\" xmlns=\"http://schemas.microsoft.com/windows/2004/02/mit/task\">\r\n" +
		"  <Principals>\r\n    <Principal id=\"Author\">\r\n" +
		"      <LogonType>InteractiveToken</LogonType>\r\n      <RunLevel>HighestAvailable</RunLevel>\r\n" +
		"    </Principal>\r\n  </Principals>\r\n" +
		"  <Settings>\r\n" +
		fmt.Sprintf("    <MultipleInstancesPolicy>%s</MultipleInstancesPolicy>\r\n", multiPolicy) +
		"    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>\r\n" +
		"    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>\r\n" +
		"    <AllowHardTerminate>false</AllowHardTerminate>\r\n" +
		"    <StartWhenAvailable>true</StartWhenAvailable>\r\n" +
		"    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>\r\n" +
		"    <Priority>5</Priority>\r\n" +
		"  </Settings>\r\n" +
		triggerBlock +
		"  <Actions Context=\"Author\">\r\n    <Exec>\r\n" +
		fmt.Sprintf("      <Command>%s</Command>%s\r\n", html.EscapeString(exe), argLine) +
		fmt.Sprintf("      <WorkingDirectory>%s</WorkingDirectory>\r\n", html.EscapeString(filepath.Dir(exe))) +
		"    </Exec>\r\n  </Actions>\r\n</Task>\r\n"
}

func registerTaskWithXML(taskName, xmlBody string) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	xmlPath := filepath.Join(dir, strings.ToLower(taskName)+".xml")
	if err := os.WriteFile(xmlPath, encodeUTF16LE(xmlBody), 0o600); err != nil {
		return err
	}
	defer os.Remove(xmlPath)

	out, err := runSchtasks("/Create", "/TN", taskName, "/XML", xmlPath, "/F")
	if err != nil {
		return fmt.Errorf("schtasks create %s falhou: %v (%s)", taskName, err, out)
	}
	return nil
}

// IsSkipUACTaskEnabled verifica se a tarefa agendada de logon elevado existe.
func IsSkipUACTaskEnabled() bool {
	_, err := runSchtasks("/Query", "/TN", skipUACTask)
	return err == nil
}

// RunSkipUACTask inicia a tarefa agendada de inicializacao elevada sem prompt UAC.
func RunSkipUACTask() error {
	out, err := runSchtasks("/Run", "/TN", skipUACTask)
	if err != nil {
		return fmt.Errorf("schtasks run %s falhou: %v (%s)", skipUACTask, err, out)
	}
	return nil
}

// RunInteractiveTask inicia o RamKiller elevado em janela visivel via Tarefa Agendada.
func RunInteractiveTask() error {
	out, err := runSchtasks("/Run", "/TN", interactiveTask)
	if err != nil {
		return fmt.Errorf("schtasks run %s falhou: %v (%s)", interactiveTask, err, out)
	}
	return nil
}

// SetSkipUACTask cria ou remove a tarefa agendada de logon elevado resistente a bateria.
func SetSkipUACTask(enabled bool, minimized bool) error {
	_, _ = runSchtasks("/Delete", "/TN", legacyUACTask, "/F")
	if !enabled {
		out, err := runSchtasks("/Delete", "/TN", skipUACTask, "/F")
		if err != nil && IsSkipUACTaskEnabled() {
			return fmt.Errorf("schtasks delete falhou: %v (%s)", err, out)
		}
		return nil
	}
	exe, err := validatedExePath()
	if err != nil {
		return err
	}
	args := "--boot"
	if minimized {
		args = "--minimized --boot"
	}
	return registerTaskWithXML(skipUACTask, buildTaskXML(exe, args, true, "IgnoreNew"))
}

// SyncInteractiveTask registra tarefa sob demanda sem UAC para abertura manual via Menu Iniciar.
func SyncInteractiveTask() error {
	exe, err := validatedExePath()
	if err != nil {
		return err
	}
	return registerTaskWithXML(interactiveTask, buildTaskXML(exe, "", false, "Parallel"))
}
