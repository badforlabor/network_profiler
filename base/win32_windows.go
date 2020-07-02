package base

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modwtsapi32 *windows.LazyDLL = windows.NewLazySystemDLL("wtsapi32.dll")
	modkernel32 *windows.LazyDLL = windows.NewLazySystemDLL("kernel32.dll")
	modadvapi32 *windows.LazyDLL = windows.NewLazySystemDLL("advapi32.dll")
	moduserenv  *windows.LazyDLL = windows.NewLazySystemDLL("userenv.dll")
	moduser32   *windows.LazyDLL = windows.NewLazySystemDLL("user32.dll")

	procWTSEnumerateSessionsW        *windows.LazyProc = modwtsapi32.NewProc("WTSEnumerateSessionsW")
	procWTSGetActiveConsoleSessionId *windows.LazyProc = modkernel32.NewProc("WTSGetActiveConsoleSessionId")
	procWTSQueryUserToken            *windows.LazyProc = modwtsapi32.NewProc("WTSQueryUserToken")
	procDuplicateTokenEx             *windows.LazyProc = modadvapi32.NewProc("DuplicateTokenEx")
	procCreateEnvironmentBlock       *windows.LazyProc = moduserenv.NewProc("CreateEnvironmentBlock")
	procCreateProcessAsUser          *windows.LazyProc = modadvapi32.NewProc("CreateProcessAsUserW")
	getConsoleWindow                 *windows.LazyProc = modkernel32.NewProc("GetConsoleWindow")
	getWindowThreadProcessId         *windows.LazyProc = moduser32.NewProc("GetWindowThreadProcessId")
	getCurrentProcessId              *windows.LazyProc = modkernel32.NewProc("GetCurrentProcessId")
	showWindowAsync                  *windows.LazyProc = moduser32.NewProc("ShowWindowAsync")
)

const (
	WTS_CURRENT_SERVER_HANDLE uintptr = 0
)

type (
	HANDLE                 uintptr
	WTS_CONNECTSTATE_CLASS int
	HWND                   HANDLE
	DWORD                  uint32
)

const (
	WTSActive WTS_CONNECTSTATE_CLASS = iota
	WTSConnected
	WTSConnectQuery
	WTSShadow
	WTSDisconnected
	WTSIdle
	WTSListen
	WTSReset
	WTSDown
	WTSInit
)

type SECURITY_IMPERSONATION_LEVEL int

const (
	SecurityAnonymous SECURITY_IMPERSONATION_LEVEL = iota
	SecurityIdentification
	SecurityImpersonation
	SecurityDelegation
)

type TOKEN_TYPE int

const (
	TokenPrimary TOKEN_TYPE = iota + 1
	TokenImpersonazion
)

const (
	SW_HIDE            int = 0
	SW_SHOWNORMAL         = 1
	SW_NORMAL             = 1
	SW_SHOWMINIMIZED      = 2
	SW_SHOWMAXIMIZED      = 3
	SW_MAXIMIZE           = 3
	SW_SHOWNOACTIVATE     = 4
	SW_SHOW               = 5
	SW_MINIMIZE           = 6
	SW_SHOWMINNOACTIVE    = 7
	SW_SHOWNA             = 8
	SW_RESTORE            = 9
	SW_SHOWDEFAULT        = 10
	SW_MAX                = 1
)

type WTS_SESSION_INFO struct {
	SessionID      windows.Handle
	WinStationName *uint16
	State          WTS_CONNECTSTATE_CLASS
}

const (
	CREATE_UNICODE_ENVIRONMENT uint16 = 0x00000400
	CREATE_NO_WINDOW                  = 0x08000000
	CREATE_NEW_CONSOLE                = 0x00000010
	DETACHED_PROCESS                  = 0x00000008
)

// GetCurrentUserSessionId will attempt to resolve
// the session ID of the user currently active on
// the system.
func GetCurrentUserSessionId() (windows.Handle, error) {
	sessionList, err := WTSEnumerateSessions()
	if err != nil {
		return 0xFFFFFFFF, fmt.Errorf("get current user session token: %s", err)
	}

	for i := range sessionList {
		if sessionList[i].State == WTSActive {
			return sessionList[i].SessionID, nil
		}
	}

	if sessionId, _, err := procWTSGetActiveConsoleSessionId.Call(); sessionId == 0xFFFFFFFF {
		return 0xFFFFFFFF, fmt.Errorf("get current user session token: call native WTSGetActiveConsoleSessionId: %s", err)
	} else {
		return windows.Handle(sessionId), nil
	}
}

// WTSEnumerateSession will call the native
// version for Windows and parse the result
// to a Golang friendly version
func WTSEnumerateSessions() ([]*WTS_SESSION_INFO, error) {
	var (
		sessionInformation windows.Handle      = windows.Handle(0)
		sessionCount       int                 = 0
		sessionList        []*WTS_SESSION_INFO = make([]*WTS_SESSION_INFO, 0)
	)

	if returnCode, _, err := procWTSEnumerateSessionsW.Call(WTS_CURRENT_SERVER_HANDLE, 0, 1, uintptr(unsafe.Pointer(&sessionInformation)), uintptr(unsafe.Pointer(&sessionCount))); returnCode == 0 {
		return nil, fmt.Errorf("call native WTSEnumerateSessionsW: %s", err)
	}

	structSize := unsafe.Sizeof(WTS_SESSION_INFO{})
	current := uintptr(sessionInformation)
	for i := 0; i < sessionCount; i++ {
		sessionList = append(sessionList, (*WTS_SESSION_INFO)(unsafe.Pointer(current)))
		current += structSize
	}

	return sessionList, nil
}

// DuplicateUserTokenFromSessionID will attempt
// to duplicate the user token for the user logged
// into the provided session ID
func DuplicateUserTokenFromSessionID(sessionId windows.Handle) (windows.Token, error) {
	var (
		impersonationToken windows.Handle = 0
		userToken          windows.Token  = 0
	)

	if returnCode, _, err := procWTSQueryUserToken.Call(uintptr(sessionId), uintptr(unsafe.Pointer(&impersonationToken))); returnCode == 0 {
		return 0xFFFFFFFF, fmt.Errorf("call native WTSQueryUserToken: %s", err)
	}

	if returnCode, _, err := procDuplicateTokenEx.Call(uintptr(impersonationToken), 0, 0, uintptr(SecurityImpersonation), uintptr(TokenPrimary), uintptr(unsafe.Pointer(&userToken))); returnCode == 0 {
		return 0xFFFFFFFF, fmt.Errorf("call native DuplicateTokenEx: %s", err)
	}

	if err := windows.CloseHandle(impersonationToken); err != nil {
		return 0xFFFFFFFF, fmt.Errorf("close windows handle used for token duplication: %s", err)
	}

	return userToken, nil
}

func StartProcessAsCurrentUser(appPath, cmdLine, workDir string) error {
	var (
		sessionId windows.Handle
		userToken windows.Token
		envInfo   windows.Handle

		startupInfo windows.StartupInfo
		processInfo windows.ProcessInformation

		commandLine uintptr = 0
		workingDir  uintptr = 0

		err error
	)

	if sessionId, err = GetCurrentUserSessionId(); err != nil {
		return err
	}

	if userToken, err = DuplicateUserTokenFromSessionID(sessionId); err != nil {
		return fmt.Errorf("get duplicate user token for current user session: %s", err)
	}

	if returnCode, _, err := procCreateEnvironmentBlock.Call(uintptr(unsafe.Pointer(&envInfo)), uintptr(userToken), 0); returnCode == 0 {
		return fmt.Errorf("create environment details for process: %s", err)
	}

	creationFlags := CREATE_UNICODE_ENVIRONMENT | CREATE_NEW_CONSOLE
	startupInfo.ShowWindow = SW_SHOW
	startupInfo.Desktop = windows.StringToUTF16Ptr("winsta0\\default")

	if len(cmdLine) > 0 {
		commandLine = uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(cmdLine)))
	}
	if len(workDir) > 0 {
		workingDir = uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(workDir)))
	}

	// TODO:增加环境变量的设置
	//syscall.Environ()
	if returnCode, _, err := procCreateProcessAsUser.Call(
		uintptr(userToken), uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(appPath))), commandLine, 0, 0, 0,
		uintptr(creationFlags), uintptr(envInfo), workingDir, uintptr(unsafe.Pointer(&startupInfo)), uintptr(unsafe.Pointer(&processInfo)),
	); returnCode == 0 {
		return fmt.Errorf("create process as user: %s", err)
	}

	return nil
}

// 控制CMD控制台的显示
func GetConsoleWindow() HWND {
	ret, _, _ := getConsoleWindow.Call()
	return HWND(ret)
}
func GetWindowThreadProcessId(hwnd HWND) (HANDLE, DWORD) {
	var processId DWORD
	ret, _, _ := getWindowThreadProcessId.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&processId)),
	)
	return HANDLE(ret), processId
}
func GetCurrentProcessId() DWORD {
	id, _, _ := getCurrentProcessId.Call()
	return DWORD(id)
}
func ShowWindowAsync(hwnd HWND, cmdshow int) bool {
	ret, _, _ := showWindowAsync.Call(
		uintptr(hwnd),
		uintptr(cmdshow),
	)
	return ret != 0
}
func ShowConsoleAsync(commandShow int) {
	console := GetConsoleWindow()
	if console != 0 {
		_, consoleProcID := GetWindowThreadProcessId(console)
		if GetCurrentProcessId() == consoleProcID {
			ShowWindowAsync(console, commandShow)
		}
	}
}
