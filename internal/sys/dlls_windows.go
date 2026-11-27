//go:build windows

package sys

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func systemRoot() string {
	for _, key := range []string{"SystemRoot", "WINDIR"} {
		if val := strings.TrimSpace(os.Getenv(key)); val != "" {
			return filepath.Clean(val)
		}
	}
	return `C:\Windows`
}

// System32Path resolves a system binary or DLL inside %SystemRoot%\System32 (CWE-427).
func System32Path(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return filepath.Join(systemRoot(), "System32")
	}
	return filepath.Join(systemRoot(), "System32", base)
}

// LoadSystemDLL loads a Windows DLL exclusively from System32 to prevent DLL hijacking.
func LoadSystemDLL(name string) *syscall.LazyDLL {
	return syscall.NewLazyDLL(System32Path(name))
}

var (
	Kernel32 = LoadSystemDLL("kernel32.dll")
	Psapi    = LoadSystemDLL("psapi.dll")
	Ntdll    = LoadSystemDLL("ntdll.dll")
	Advapi32 = LoadSystemDLL("advapi32.dll")
	User32   = LoadSystemDLL("user32.dll")
	Shell32  = LoadSystemDLL("shell32.dll")
	Ole32    = LoadSystemDLL("ole32.dll")
	Gdi32    = LoadSystemDLL("gdi32.dll")
	Gdiplus  = LoadSystemDLL("gdiplus.dll")
	Dwmapi   = LoadSystemDLL("dwmapi.dll")
	Comdlg32 = LoadSystemDLL("comdlg32.dll")
	Iphlpapi = LoadSystemDLL("iphlpapi.dll")
)

var (
	procGlobalMemoryStatusEx      = Kernel32.NewProc("GlobalMemoryStatusEx")
	procSetSystemFileCacheSize    = Kernel32.NewProc("SetSystemFileCacheSize")
	procGetLogicalDrives          = Kernel32.NewProc("GetLogicalDrives")
	procGetDriveTypeW             = Kernel32.NewProc("GetDriveTypeW")
	procCreateFileW               = Kernel32.NewProc("CreateFileW")
	procFlushFileBuffers          = Kernel32.NewProc("FlushFileBuffers")
	procCloseHandle               = Kernel32.NewProc("CloseHandle")
	procGetCurrentProcess         = Kernel32.NewProc("GetCurrentProcess")
	procOpenProcess               = Kernel32.NewProc("OpenProcess")
	procTerminateProcess          = Kernel32.NewProc("TerminateProcess")
	procSetProcessWorkingSetEx    = Kernel32.NewProc("SetProcessWorkingSetSizeEx")
	procProcessIdToSessionId      = Kernel32.NewProc("ProcessIdToSessionId")
	procGetProcessTimes           = Kernel32.NewProc("GetProcessTimes")
	procGetTickCount64            = Kernel32.NewProc("GetTickCount64")
	procCreateToolhelp32Snapshot  = Kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW           = Kernel32.NewProc("Process32FirstW")
	procProcess32NextW            = Kernel32.NewProc("Process32NextW")
	procCreateEventW              = Kernel32.NewProc("CreateEventW")
	procOpenEventW                = Kernel32.NewProc("OpenEventW")
	procSetEvent                  = Kernel32.NewProc("SetEvent")
	procResetEvent                = Kernel32.NewProc("ResetEvent")
	procOpenMutexW                = Kernel32.NewProc("OpenMutexW")
	procWaitForSingleObject       = Kernel32.NewProc("WaitForSingleObject")

	procK32GetPerformanceInfo   = Kernel32.NewProc("K32GetPerformanceInfo")
	procK32GetProcessMemoryInfo = Kernel32.NewProc("K32GetProcessMemoryInfo")
	procK32EmptyWorkingSet      = Kernel32.NewProc("K32EmptyWorkingSet")
	procPsapiGetPerformanceInfo = Psapi.NewProc("GetPerformanceInfo")
	procPsapiGetProcMemInfo     = Psapi.NewProc("GetProcessMemoryInfo")
	procPsapiEmptyWorkingSet    = Psapi.NewProc("EmptyWorkingSet")

	procNtQuerySystemInformation  = Ntdll.NewProc("NtQuerySystemInformation")
	procNtSetSystemInformation    = Ntdll.NewProc("NtSetSystemInformation")
	procNtQueryInformationProcess = Ntdll.NewProc("NtQueryInformationProcess")

	procOpenProcessToken      = Advapi32.NewProc("OpenProcessToken")
	procGetTokenInformation   = Advapi32.NewProc("GetTokenInformation")
	procLookupPrivilegeValueW = Advapi32.NewProc("LookupPrivilegeValueW")
	procAdjustTokenPrivileges = Advapi32.NewProc("AdjustTokenPrivileges")
	procRegOpenKeyExW         = Advapi32.NewProc("RegOpenKeyExW")
	procRegQueryValueExW      = Advapi32.NewProc("RegQueryValueExW")
	procRegSetValueExW        = Advapi32.NewProc("RegSetValueExW")
	procRegDeleteValueW       = Advapi32.NewProc("RegDeleteValueW")
	procRegCloseKey           = Advapi32.NewProc("RegCloseKey")

	procGetForegroundWindow      = User32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = User32.NewProc("GetWindowThreadProcessId")
	procOpenDesktopW             = User32.NewProc("OpenDesktopW")
	procCloseDesktop             = User32.NewProc("CloseDesktop")
	procEnumDesktopWindows       = User32.NewProc("EnumDesktopWindows")
	procEnumWindows              = User32.NewProc("EnumWindows")
	procEnumChildWindows         = User32.NewProc("EnumChildWindows")
	procIsWindowVisible          = User32.NewProc("IsWindowVisible")
	procGetWindowTextLengthW     = User32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW           = User32.NewProc("GetWindowTextW")
	procDwmGetWindowAttribute    = Dwmapi.NewProc("DwmGetWindowAttribute")

	procShellExecuteW               = Shell32.NewProc("ShellExecuteW")
	procSHChangeNotify              = Shell32.NewProc("SHChangeNotify")
	procSHQueryUserNotificationState = Shell32.NewProc("SHQueryUserNotificationState")
	procCoInitializeEx      = Ole32.NewProc("CoInitializeEx")
	procCoUninitialize      = Ole32.NewProc("CoUninitialize")
	procCoCreateInst        = Ole32.NewProc("CoCreateInstance")
	procGetExtendedTcpTable = Iphlpapi.NewProc("GetExtendedTcpTable")
)
