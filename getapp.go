package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	moduser32                    = syscall.NewLazyDLL("user32.dll")
	modkernel32                  = syscall.NewLazyDLL("kernel32.dll")
	modpsapi                     = syscall.NewLazyDLL("psapi.dll")
	procGetForegroundWindow      = moduser32.NewProc("GetForegroundWindow")
	procGetWindowRect            = moduser32.NewProc("GetWindowRect")
	procIsZoomed                 = moduser32.NewProc("IsZoomed")
	procGetWindowThreadProcessId = moduser32.NewProc("GetWindowThreadProcessId")
	procOpenProcess              = modkernel32.NewProc("OpenProcess")
	procCloseHandle              = modkernel32.NewProc("CloseHandle")
	procGetExitCodeProcess       = modkernel32.NewProc("GetExitCodeProcess")
	procGetModuleBaseNameW       = modpsapi.NewProc("GetModuleBaseNameW")
	procGetSystemMetrics         = moduser32.NewProc("GetSystemMetrics")
)

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type Process struct {
	Name string
	pid  uint32
}

// GetFullscreenApp 最前面のフルスクリーンアプリケーション名を取得
func GetFullscreenApp() (*Process, error) {
	// フォアグラウンドウィンドウ取得
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return nil, fmt.Errorf("no foreground window")
	}

	/*
		// 最大化チェック
		isZoomed, _, _ := procIsZoomed.Call(hwnd)
		if isZoomed == 0 {
			return "", fmt.Errorf("foreground window is not maximized")
		}
	*/

	// ウィンドウ矩形取得
	var rect RECT
	_, _, _ = procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))

	// 画面サイズ取得
	screenW, _, _ := procGetSystemMetrics.Call(0) // SM_CXSCREEN
	screenH, _, _ := procGetSystemMetrics.Call(1) // SM_CYSCREEN
	// 全画面サイズチェック（タスクバー考慮せず単純比較）
	if int32(screenW) != rect.Right-rect.Left || int32(screenH) != rect.Bottom-rect.Top {
		return nil, fmt.Errorf("foreground window is not fullscreen size")
	}

	// プロセスID取得
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

	// プロセスハンドル取得
	const PROCESS_QUERY_INFORMATION = 0x0400
	const PROCESS_VM_READ = 0x0010
	hProcess, _, _ := procOpenProcess.Call(
		PROCESS_QUERY_INFORMATION|PROCESS_VM_READ,
		0,
		uintptr(pid),
	)
	if hProcess == 0 {
		return nil, fmt.Errorf("cannot open process %d", pid)
	}
	defer procCloseHandle.Call(hProcess)

	// プロセス名取得（Unicode版）
	buf := make([]uint16, 260)
	procGetModuleBaseNameW.Call(
		hProcess,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	name := syscall.UTF16ToString(buf)
	//name = name[:len(name)] // null終端除去

	return &Process{Name: name, pid: pid}, nil
}

const STILL_ACTIVE uint32 = 259

func (p *Process) IsProcessExited() (bool, error) {
	// OpenProcess(DWORD dwDesiredAccess, BOOL bInheritHandle, DWORD dwProcessId)
	h, _, err := procOpenProcess.Call(
		0x1000, // PROCESS_QUERY_LIMITED_INFORMATION
		0,      // bInheritHandle = FALSE
		uintptr(p.pid),
	)

	if h == 0 {
		if err == syscall.ERROR_PROC_NOT_FOUND || err == syscall.ERROR_ACCESS_DENIED {
			return true, nil // プロセスが存在しない=終了済み
		}
		return false, fmt.Errorf("OpenProcess failed: %v", err)
	}

	// GetExitCodeProcess(HANDLE hProcess, LPDWORD lpExitCode)
	var exitCode uint32
	ret, _, err := procGetExitCodeProcess.Call(h, uintptr(unsafe.Pointer(&exitCode)))
	if ret == 0 {
		procCloseHandle.Call(h)
		return false, fmt.Errorf("GetExitCodeProcess failed: %v", err)
	}

	procCloseHandle.Call(h)

	// 実行中ならSTILL_ACTIVE (259)、終了済みなら別値
	return exitCode != STILL_ACTIVE, nil
}
